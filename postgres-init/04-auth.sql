-- Настройка аутентификации для пользователя orders_user
ALTER USER orders_user WITH PASSWORD 'orders_pass';
ALTER USER orders_user WITH LOGIN;
ALTER USER orders_user WITH CREATEDB;
ALTER USER orders_user WITH SUPERUSER;

-- Создаем пользователя с правильным паролем
DROP USER IF EXISTS orders_user;
CREATE USER orders_user WITH PASSWORD 'orders_pass';
ALTER USER orders_user WITH LOGIN;
ALTER USER orders_user WITH CREATEDB;
ALTER USER orders_user WITH SUPERUSER;
GRANT ALL PRIVILEGES ON DATABASE orders_db TO orders_user; 