PARENT_PATH=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )
docker run -v $PARENT_PATH/migrations:/migrations --network host migrate/migrate -path=/migrations/ -database "postgres://local_user:local_pass@127.0.0.1/local_db?sslmode=disable" up 1
