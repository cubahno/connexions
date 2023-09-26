FROM cubahno/connexions:latest

# remove not needed stuff
RUN rm -rf /app/resources/data/services
RUN rm -rf /app/resources/ui

# copy your service definitions, contexts and config
COPY services /app/resources/data/services
COPY contexts /app/resources/data/contexts
COPY config.yml /app/resources/data/config.yml

# done
