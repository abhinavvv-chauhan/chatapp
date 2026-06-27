-- name: SavePushSubscription :exec
INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth)
VALUES ($1, $2, $3, $4)
ON CONFLICT (endpoint) DO UPDATE 
SET user_id = EXCLUDED.user_id, p256dh = EXCLUDED.p256dh, auth = EXCLUDED.auth;

-- name: GetSubscriptionByUsername :one
SELECT ps.endpoint, ps.p256dh, ps.auth
FROM push_subscriptions ps
JOIN users u ON ps.user_id = u.id
WHERE u.username = $1 LIMIT 1;