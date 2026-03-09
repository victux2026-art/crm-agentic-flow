-- Crear la tabla 'persons'
CREATE TABLE IF NOT EXISTS persons (
    id SERIAL PRIMARY KEY,
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    phone_number VARCHAR(50),
    company_name VARCHAR(255),
    job_title VARCHAR(255),
    linkedin_profile VARCHAR(255),
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB DEFAULT '{}',
    embedding VECTOR(1536)
);

-- Crear la tabla 'flows'
CREATE TABLE IF NOT EXISTS flows (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    person_id INT NOT NULL,
    company_name VARCHAR(255),
    status VARCHAR(50) NOT NULL,
    value NUMERIC(10, 2) NOT NULL,
    currency VARCHAR(10) NOT NULL,
    expected_close_date TIMESTAMP WITH TIME ZONE,
    actual_close_date TIMESTAMP WITH TIME ZONE,
    priority VARCHAR(50) NOT NULL,
    health_score INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB DEFAULT '{}',
    embedding VECTOR(1536),
    FOREIGN KEY (person_id) REFERENCES persons(id) ON DELETE CASCADE
);

-- Crear la tabla 'activities'
CREATE TABLE IF NOT EXISTS activities (
    id SERIAL PRIMARY KEY,
    flow_id INT,
    person_id INT,
    type VARCHAR(50) NOT NULL,
    subject VARCHAR(255),
    description TEXT,
    due_date TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB DEFAULT '{}',
    embedding VECTOR(1536),
    FOREIGN KEY (flow_id) REFERENCES flows(id) ON DELETE CASCADE,
    FOREIGN KEY (person_id) REFERENCES persons(id) ON DELETE CASCADE
);