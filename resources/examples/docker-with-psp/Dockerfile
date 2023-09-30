FROM cubahno/connexions:latest

# clean start
RUN rm -rf /app/resources/data/services

# get remote specs

# paypal
RUN \
    wget -P /app/resources/data/services/.openapi/paypal/ https://raw.githubusercontent.com/paypal/paypal-rest-api-specifications/main/openapi/payments_payment_v2.json && \
    wget -P /app/resources/data/services/.openapi/paypal/ https://raw.githubusercontent.com/paypal/paypal-rest-api-specifications/main/openapi/invoicing_v2.json && \
    wget -P /app/resources/data/services/.openapi/paypal/ https://raw.githubusercontent.com/paypal/paypal-rest-api-specifications/main/openapi/checkout_orders_v2.json

# adyen
RUN \
    wget -P /app/resources/data/services/.openapi/adyen/ https://raw.githubusercontent.com/Adyen/adyen-openapi/main/yaml/AccountService-v6.yaml && \
    wget -P /app/resources/data/services/.openapi/adyen/ https://raw.githubusercontent.com/Adyen/adyen-openapi/main/yaml/BinLookupService-v54.yaml && \
    wget -P /app/resources/data/services/.openapi/adyen/ https://raw.githubusercontent.com/Adyen/adyen-openapi/main/yaml/CheckoutService-v70.yaml && \
    wget -P /app/resources/data/services/.openapi/adyen/ https://raw.githubusercontent.com/Adyen/adyen-openapi/main/yaml/PaymentService-v68.yaml && \
    wget -P /app/resources/data/services/.openapi/adyen/ https://raw.githubusercontent.com/Adyen/adyen-openapi/main/yaml/RecurringService-v68.yaml && \
    wget -P /app/resources/data/services/.openapi/adyen/ https://raw.githubusercontent.com/Adyen/adyen-openapi/main/yaml/Webhooks-v1.yaml && \
    wget -P /app/resources/data/services/.openapi/adyen/ https://raw.githubusercontent.com/Adyen/adyen-openapi/main/yaml/FundService-v6.yaml

# stripe
RUN \
    wget -P /app/resources/data/services/.openapi/stripe/ https://raw.githubusercontent.com/stripe/openapi/master/openapi/spec3.yaml

# mobilepay
RUN \
    wget -P /app/resources/data/services/.openapi/mobilepay/ https://raw.githubusercontent.com/MobilePayDev/MobilePayDev.github.io/main/openapi/app_payments_api.yaml && \
    wget -P /app/resources/data/services/.openapi/mobilepay/ https://raw.githubusercontent.com/MobilePayDev/MobilePayDev.github.io/main/openapi/invoice.yaml && \
    wget -P /app/resources/data/services/.openapi/mobilepay/ https://raw.githubusercontent.com/MobilePayDev/MobilePayDev.github.io/main/openapi/online.yaml && \
    wget -P /app/resources/data/services/.openapi/mobilepay/ https://raw.githubusercontent.com/MobilePayDev/MobilePayDev.github.io/main/openapi/pos.yaml && \
    wget -P /app/resources/data/services/.openapi/mobilepay/ https://raw.githubusercontent.com/MobilePayDev/MobilePayDev.github.io/main/openapi/reporting.yaml && \
    wget -P /app/resources/data/services/.openapi/mobilepay/ https://raw.githubusercontent.com/MobilePayDev/MobilePayDev.github.io/main/openapi/subscriptions.yaml && \
    wget -P /app/resources/data/services/.openapi/mobilepay/ https://raw.githubusercontent.com/MobilePayDev/MobilePayDev.github.io/main/openapi/webhooks.yaml

# vipps
RUN \
    wget -P /app/resources/data/services/.openapi/vipps/ https://raw.githubusercontent.com/vippsas/vipps-recurring-api/master/docs/openapi.yaml

# copy your service definitions, contexts and config
COPY services /app/resources/data/services
COPY contexts /app/resources/data/contexts
COPY config.yml /app/resources/data/config.yml

# optimize openapi specs to replace circular references and to improve performance:
RUN simplify-schemas --src=/app/resources/data/services --replace=true

# done
