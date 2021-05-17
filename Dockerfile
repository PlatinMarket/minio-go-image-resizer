FROM alpine

# Install Gifsicle dependency.
RUN apk add gifsicle

# Install Common CA certificates.
RUN apk add ca-certificates

# Copy resizer to container.
COPY ./bin/resizer-amd64 /usr/local/bin/resizer

# Copy run script to container.
COPY ./docker/run.sh /usr/local/bin/run

# Make it executable.
RUN ["chmod", "+x", "/usr/local/bin/run"]

# Expose the resizer port.
EXPOSE 2222

# Run.
CMD [ "/usr/local/bin/run" ]
