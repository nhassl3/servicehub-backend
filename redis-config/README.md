# Redis configuration

### You need to implement configuration for Redis

    bind localhost
    port 6380
    requirepass "some-password"
    appendonly yes
    appendfsync everysec

### When you realize steps above you should run:
    make redis