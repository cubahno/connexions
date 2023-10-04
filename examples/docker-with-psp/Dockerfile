FROM cubahno/connexions:latest

# clean start
RUN rm -rf /app/resources/data/services
RUN rm -rf /app/resources/data/contexts/petstore.yml

# get remote specs and
# optimize them to replace circular references and to improve performance:

# paypal
RUN \
    wget -P /app/resources/data/services/.openapi/paypal/ https://raw.githubusercontent.com/paypal/paypal-rest-api-specifications/main/openapi/payments_payment_v2.json && \
    wget -P /app/resources/data/services/.openapi/paypal/ https://raw.githubusercontent.com/paypal/paypal-rest-api-specifications/main/openapi/invoicing_v2.json && \
    wget -P /app/resources/data/services/.openapi/paypal/ https://raw.githubusercontent.com/paypal/paypal-rest-api-specifications/main/openapi/checkout_orders_v2.json && \
    simplify-schemas --src=/app/resources/data/services/.openapi/paypal/ --replace=true

# adyen
RUN \
    wget -P /app/resources/data/services/.openapi/adyen/ https://raw.githubusercontent.com/Adyen/adyen-openapi/main/yaml/AccountService-v6.yaml && \
    wget -P /app/resources/data/services/.openapi/adyen/ https://raw.githubusercontent.com/Adyen/adyen-openapi/main/yaml/BinLookupService-v54.yaml && \
    wget -P /app/resources/data/services/.openapi/adyen/ https://raw.githubusercontent.com/Adyen/adyen-openapi/main/yaml/CheckoutService-v70.yaml && \
    wget -P /app/resources/data/services/.openapi/adyen/ https://raw.githubusercontent.com/Adyen/adyen-openapi/main/yaml/PaymentService-v68.yaml && \
    wget -P /app/resources/data/services/.openapi/adyen/ https://raw.githubusercontent.com/Adyen/adyen-openapi/main/yaml/RecurringService-v68.yaml && \
    wget -P /app/resources/data/services/.openapi/adyen/ https://raw.githubusercontent.com/Adyen/adyen-openapi/main/yaml/Webhooks-v1.yaml && \
    wget -P /app/resources/data/services/.openapi/adyen/ https://raw.githubusercontent.com/Adyen/adyen-openapi/main/yaml/FundService-v6.yaml && \
    simplify-schemas --src=/app/resources/data/services/.openapi/adyen/ --replace=true

# stripe
RUN \
    wget -P /app/resources/data/services/.openapi/stripe/ https://raw.githubusercontent.com/stripe/openapi/master/openapi/spec3.yaml && \
    simplify-schemas --src=/app/resources/data/services/.openapi/stripe/ --replace=true --only-required=true

# mobilepay
RUN \
    wget -P /app/resources/data/services/.openapi/mobilepay/ https://raw.githubusercontent.com/MobilePayDev/MobilePayDev.github.io/main/openapi/app_payments_api.yaml && \
    wget -P /app/resources/data/services/.openapi/mobilepay/ https://raw.githubusercontent.com/MobilePayDev/MobilePayDev.github.io/main/openapi/invoice.yaml && \
    wget -P /app/resources/data/services/.openapi/mobilepay/ https://raw.githubusercontent.com/MobilePayDev/MobilePayDev.github.io/main/openapi/online.yaml && \
    wget -P /app/resources/data/services/.openapi/mobilepay/ https://raw.githubusercontent.com/MobilePayDev/MobilePayDev.github.io/main/openapi/pos.yaml && \
    wget -P /app/resources/data/services/.openapi/mobilepay/ https://raw.githubusercontent.com/MobilePayDev/MobilePayDev.github.io/main/openapi/reporting.yaml && \
    wget -P /app/resources/data/services/.openapi/mobilepay/ https://raw.githubusercontent.com/MobilePayDev/MobilePayDev.github.io/main/openapi/subscriptions.yaml && \
    wget -P /app/resources/data/services/.openapi/mobilepay/ https://raw.githubusercontent.com/MobilePayDev/MobilePayDev.github.io/main/openapi/webhooks.yaml && \
    simplify-schemas --src=/app/resources/data/services/.openapi/mobilepay/ --replace=true

# vipps
RUN \
    wget -P /app/resources/data/services/.openapi/vipps/ https://raw.githubusercontent.com/vippsas/vipps-recurring-api/master/docs/openapi.yaml && \
    simplify-schemas --src=/app/resources/data/services/.openapi/vipps/ --replace=true

# copy your service definitions, contexts and config
COPY services /app/resources/data/services
COPY contexts /app/resources/data/contexts
COPY config.yml /app/resources/data/config.yml

# done
