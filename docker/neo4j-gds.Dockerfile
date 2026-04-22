ARG NEO4J_BASE_IMAGE=docker.io/library/neo4j:4.4.42
FROM ${NEO4J_BASE_IMAGE}

ARG GDS_VERSION=2.6.8
ARG GDS_URL=https://github.com/neo4j/graph-data-science/releases/download/${GDS_VERSION}/neo4j-graph-data-science-${GDS_VERSION}.jar

USER root

RUN set -eux; \
    mkdir -p /plugins; \
    wget -O /plugins/graph-data-science.jar "${GDS_URL}"; \
    chown -R neo4j:neo4j /plugins; \
    chmod 0644 /plugins/graph-data-science.jar

USER neo4j
