-- Настройка аутентификации для пользователя orders_user
ALTER USER orders_user WITH PASSWORD 'orders_pass';
ALTER USER orders_user WITH LOGIN;
ALTER USER orders_user WITH CREATEDB;
ALTER USER orders_user WITH SUPERUSER; 