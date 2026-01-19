package consts

// Redis connectivity messages
const (
	RedisConnectSuccess    = "Successfully connected to Redis."
	RedisConnectFailure    = "Failed to connect to Redis."
	RedisDisconnectSuccess = "Successfully disconnected from Redis."
	RedisDisconnectFailure = "Failed to disconnect from Redis."
	RedisPingSuccess       = "Redis ping successful."
	RedisPingFailure       = "Redis ping failed."
)

// Redis operation messages
const (
	RedisSetSuccess       = "Value set in Redis successfully."
	RedisSetFailure       = "Failed to set value in Redis."
	RedisGetSuccess       = "Value retrieved from Redis successfully."
	RedisGetFailure       = "Failed to get value from Redis."
	RedisDeleteSuccess    = "Key deleted from Redis successfully."
	RedisDeleteFailure    = "Failed to delete key from Redis."
	RedisExistsTrue       = "Key exists in Redis."
	RedisExistsFalse      = "Key does not exist in Redis."
	RedisExpireSet        = "Expiration set for Redis key."
	RedisExpireFailure    = "Failed to set expiration for Redis key."
	RedisOperationSuccess = "Redis operation completed successfully."
	RedisOperationFailure = "Redis operation failed."
	RedisSaveLoanSuccess  = "Loan saved in Redis successfully."
	RedisGetLoanSuccess   = "Loan retrieved from Redis successfully."
)
