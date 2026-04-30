FROM php:8.4-apache

RUN apt-get update && apt-get install -y \
    git unzip libzip-dev librdkafka-dev \
    && docker-php-ext-install zip pdo pdo_mysql \
    && pecl install rdkafka \
    && docker-php-ext-enable rdkafka \
    && apt-get clean

COPY --from=composer:latest /usr/bin/composer /usr/bin/composer

RUN echo '<VirtualHost *:80>\n\
    DocumentRoot /var/www/html/public\n\
    <Directory /var/www/html/public>\n\
        AllowOverride All\n\
        Require all granted\n\
    </Directory>\n\
</VirtualHost>' > /etc/apache2/sites-available/000-default.conf \
    && a2enmod rewrite

WORKDIR /var/www/html

COPY composer.json composer.lock* ./
RUN composer install --no-interaction --no-scripts

COPY . .

RUN chown -R www-data:www-data /var/www/html