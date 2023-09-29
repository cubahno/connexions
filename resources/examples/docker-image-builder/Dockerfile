FROM cubahno/connexions:latest

# remove not needed stuff
RUN rm -rf /app/resources/data/services
RUN rm -rf /app/resources/ui

# copy your service definitions, contexts and config
COPY services /app/resources/data/services
COPY contexts /app/resources/data/contexts
COPY config.yml /app/resources/data/config.yml

# optimize openapi specs for better performance:
# - removes circular references
# - removes unused properties
# - merges all $refs
RUN make simplify-schemas src=/app/resources/data/services replace=true

# done
