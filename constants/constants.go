package constants

// Environment variable names
const (
	HEALTH_TOKEN             = "HEALTH_TOKEN"
	ALLOWED_ORIGINS          = "ALLOWED_ORIGINS"
	PYROSCOPE_SERVER_ADDRESS = "PYROSCOPE_SERVER_ADDRESS"
	PYROSCOPE_TENANT_ID      = "PYROSCOPE_TENANT_ID"

	// CACHE_TTLS maps route group to TTL, e.g. "library=30m,wishlist=15m".
	// Route groups not listed fall back to CACHE_DEFAULT_TTL.
	CACHE_TTLS        = "CACHE_TTLS"
	CACHE_DEFAULT_TTL = "CACHE_DEFAULT_TTL"

	// DISTRIBUTED_RATE_LIMIT_ENABLED toggles the Redis-backed per-client limiter on in
	// place of the process-wide golang.org/x/time/rate limiter. Requires REDIS_HOST.
	DISTRIBUTED_RATE_LIMIT_ENABLED = "DISTRIBUTED_RATE_LIMIT_ENABLED"
	RATE_LIMIT_CHEAP               = "RATE_LIMIT_CHEAP"
	RATE_LIMIT_CHEAP_WINDOW        = "RATE_LIMIT_CHEAP_WINDOW_SECONDS"
	RATE_LIMIT_STANDARD            = "RATE_LIMIT_STANDARD"
	RATE_LIMIT_STANDARD_WINDOW     = "RATE_LIMIT_STANDARD_WINDOW_SECONDS"

	// AUTH_API_URL points at auth-api's base URL, used to verify bearer tokens and resolve
	// the caller's subject (user ID) via POST /authz/check.
	AUTH_API_URL = "AUTH_API_URL"
)

// Value constants
const (
	ServiceName = "game-room-api"

	// ProfilingEnabledFlag is the feature-flag key gating continuous profiling, evaluated via
	// api-core.go/featureflags.
	ProfilingEnabledFlag = "profiling-enabled"

	ErrorRateLimitUnavailable = "rate_limit_unavailable"
	ErrorForbidden            = "forbidden"
)
