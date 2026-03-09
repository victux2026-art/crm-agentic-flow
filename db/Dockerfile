FROM postgres:16

# Instalar las herramientas necesarias para construir pgvector
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    git \
    postgresql-server-dev-16 \
    ca-certificates

RUN update-ca-certificates

# Clonar el repositorio de pgvector (ignorando verificación SSL para este paso crucial)
RUN GIT_SSL_NOVERIFY=true git clone --branch v0.6.0 https://github.com/pgvector/pgvector.git /usr/src/pgvector

# Construir e instalar pgvector, forzando la instalación en el directorio de PostgreSQL
WORKDIR /usr/src/pgvector
RUN make && make install PG_CONFIG=/usr/bin/pg_config

# Limpiar archivos de compilación
RUN apt-get remove -y build-essential git postgresql-server-dev-16 ca-certificates \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* /usr/src/pgvector
