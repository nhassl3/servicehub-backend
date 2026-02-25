package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/nhassl3/servicehub/internal/db"
	"github.com/nhassl3/servicehub/internal/domain"
)

type OrderRepo struct {
	store *db.Store
}

func NewOrderRepo(store *db.Store) *OrderRepo {
	return &OrderRepo{store: store}
}

func (r *OrderRepo) Create(ctx context.Context, username string) (*domain.Order, error) {
	row, err := r.store.CreateOrder(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("order_repo.Create: %w", err)
	}
	return mapOrder(row), nil
}

func (r *OrderRepo) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	uid, err := parseUUID(id)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	row, err := r.store.GetOrderByID(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("order_repo.GetByID: %w", err)
	}
	order := mapOrder(row)
	order.Items, _ = r.getItems(ctx, row.ID)
	return order, nil
}

func (r *OrderRepo) GetByUID(ctx context.Context, uid string) (*domain.Order, error) {
	u, err := parseUUID(uid)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	row, err := r.store.GetOrderByUID(ctx, u)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("order_repo.GetByUID: %w", err)
	}
	return mapOrder(row), nil
}

func (r *OrderRepo) List(ctx context.Context, params domain.ListOrdersParams) ([]domain.Order, int64, error) {
	total, err := r.store.CountOrdersByUsername(ctx, db.CountOrdersByUsernameParams{
		Username:     params.Username,
		FilterStatus: params.Status,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("order_repo.List count: %w", err)
	}

	rows, err := r.store.ListOrdersByUsername(ctx, db.ListOrdersByUsernameParams{
		Username:     params.Username,
		FilterStatus: params.Status,
		Limit:        params.Limit,
		Offset:       params.Offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("order_repo.List: %w", err)
	}

	orders := make([]domain.Order, len(rows))
	for i, row := range rows {
		orders[i] = *mapOrder(row)
	}
	return orders, total, nil
}

func (r *OrderRepo) UpdateStatus(ctx context.Context, id, status string) (*domain.Order, error) {
	uid, err := parseUUID(id)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	row, err := r.store.UpdateOrderStatus(ctx, db.UpdateOrderStatusParams{
		ID:     uid,
		Status: status,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("order_repo.UpdateStatus: %w", err)
	}
	return mapOrder(row), nil
}

// Checkout performs a full transactional checkout:
//  1. Get cart + items
//  2. Check balance (FOR UPDATE lock)
//  3. Create order
//  4. Insert order items + increment product sales_count
//  5. Deduct balance + record transaction
//  6. Clear cart
//  7. Mark order as paid
func (r *OrderRepo) Checkout(ctx context.Context, username string) (*domain.Order, error) {
	var finalOrder *domain.Order

	err := r.store.ExecTx(ctx, func(q *db.Queries) error {
		// 1. Get or verify cart
		cart, err := q.UpsertCart(ctx, username)
		if err != nil {
			return fmt.Errorf("get cart: %w", err)
		}

		cartItems, err := q.GetCartItems(ctx, cart.ID)
		if err != nil {
			return fmt.Errorf("get cart items: %w", err)
		}
		if len(cartItems) == 0 {
			return domain.ErrEmptyCart
		}

		var totalAmount float64
		for _, ci := range cartItems {
			totalAmount += float64(ci.Quantity) * ci.UnitPrice
		}

		// 2. Check balance
		bal, err := q.GetBalanceForUpdate(ctx, username)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrInsufficientFunds
			}
			return fmt.Errorf("get balance: %w", err)
		}
		if bal.Amount < totalAmount {
			return domain.ErrInsufficientFunds
		}

		// 3. Create order
		order, err := q.CreateOrder(ctx, username)
		if err != nil {
			return fmt.Errorf("create order: %w", err)
		}

		// 4. Insert items + increment product sales
		for _, ci := range cartItems {
			if _, err = q.CreateOrderItem(ctx, db.CreateOrderItemParams{
				OrderID:   order.ID,
				ProductID: ci.ProductID,
				Quantity:  ci.Quantity,
				UnitPrice: ci.UnitPrice,
			}); err != nil {
				return fmt.Errorf("create order item: %w", err)
			}
			if err = q.IncrementProductSales(ctx, db.IncrementProductSalesParams{
				ID:         ci.ProductID,
				SalesCount: ci.Quantity,
			}); err != nil {
				return fmt.Errorf("increment sales: %w", err)
			}
		}

		// 5. Deduct balance + record
		if _, err = q.DeductFromBalance(ctx, db.DeductFromBalanceParams{
			Username: username,
			Amount:   totalAmount,
		}); err != nil {
			return fmt.Errorf("deduct balance: %w", err)
		}
		if _, err = q.CreateBalanceTx(ctx, db.CreateBalanceTxParams{
			Username: username,
			Type:     domain.TxTypeWithdraw,
			Amount:   totalAmount,
			Comment:  fmt.Sprintf("Order %s", order.ID),
		}); err != nil {
			return fmt.Errorf("record balance tx: %w", err)
		}

		// 6. Clear cart
		if err = q.ClearCart(ctx, cart.ID); err != nil {
			return fmt.Errorf("clear cart: %w", err)
		}

		// 7. Update total + mark paid
		paidOrder, err := q.UpdateOrderTotal(ctx, db.UpdateOrderTotalParams{
			ID:          order.ID,
			TotalAmount: totalAmount,
		})
		if err != nil {
			return fmt.Errorf("update total: %w", err)
		}
		paidOrder, err = q.UpdateOrderStatus(ctx, db.UpdateOrderStatusParams{
			ID:     paidOrder.ID,
			Status: domain.OrderStatusPaid,
		})
		if err != nil {
			return fmt.Errorf("update status: %w", err)
		}

		finalOrder = mapOrder(paidOrder)

		// Load items for the response
		dbItems, err := q.GetOrderItems(ctx, paidOrder.ID)
		if err == nil {
			finalOrder.Items = mapOrderItems(dbItems)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("order_repo.Checkout: %w", err)
	}
	return finalOrder, nil
}

// ── Internal helpers ─────────────────────────────────────────────────────────

func (r *OrderRepo) getItems(ctx context.Context, orderID uuid.UUID) ([]domain.OrderItem, error) {
	rows, err := r.store.GetOrderItems(ctx, orderID)
	if err != nil {
		return nil, err
	}
	return mapOrderItems(rows), nil
}

func mapOrderItems(rows []db.OrderItem) []domain.OrderItem {
	items := make([]domain.OrderItem, len(rows))
	for i, row := range rows {
		items[i] = domain.OrderItem{
			ID:         row.ID,
			OrderID:    row.OrderID.String(),
			ProductID:  row.ProductID.String(),
			Quantity:   int(row.Quantity),
			UnitPrice:  row.UnitPrice,
			TotalPrice: numericToFloat64(row.TotalPrice),
		}
	}
	return items
}

// ── Mapping ──────────────────────────────────────────────────────────────────

func mapOrder(o db.Order) *domain.Order {
	return &domain.Order{
		ID:          o.ID.String(),
		UID:         o.Uid.String(),
		Username:    o.Username,
		Status:      o.Status,
		TotalAmount: o.TotalAmount,
		CreatedAt:   pgTimeTZ(o.CreatedAt, time.UTC),
		UpdatedAt:   pgTimeTZ(o.UpdatedAt, time.UTC),
	}
}
