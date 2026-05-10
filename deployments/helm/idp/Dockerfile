FROM ghcr.io/asgardeo/thunder:0.37.0

# Switch to root temporarily to modify permissions
USER root

# Grant Group 0 ownership and mirror User permissions to the Group
RUN mkdir -p /opt/thunder/repository/certs && \
    chgrp -R 0 /opt/thunder/repository/resources/security /opt/thunder/repository/certs && \
    chmod -R g=u /opt/thunder/repository/resources/security /opt/thunder/repository/certs

# Switch back to the default non-root user
USER 1001
