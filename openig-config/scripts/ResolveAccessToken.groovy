import org.forgerock.http.oauth2.AccessTokenInfo
import org.forgerock.json.JsonValue

import org.forgerock.json.jose.builders.JwtBuilderFactory
import org.forgerock.json.jose.jws.SignedJwt

def httpRequest = new Request()
httpRequest.method = "GET"
httpRequest.uri = config.tokenInfoEndpoint.asString()
httpRequest.headers['Authorization'] = "OAuth " + token
def response = http.send(httpRequest).get(5, java.util.concurrent.TimeUnit.SECONDS)
try {
    if(response.status.code != 200) {
        throw new Exception("error getting user info " + response.status)
    }
    def sjwt = new JwtBuilderFactory().reconstruct(response.entity.string, SignedJwt.class)
    logger.info("sjwt: " + sjwt.getClaimsSet())
    return new AccessTokenInfo(new JsonValue(sjwt.getClaimsSet().getProperties().all), token, new HashSet<>(), sjwt.getClaimsSet().getExpirationTime().getTime() * 1000)
} catch(Exception ex) {
    logger.warn("exception occurred: " + ex + ", response " + response.entity)
    throw ex
} finally {
    response.close()
}