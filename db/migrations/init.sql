-- SQL migration file to initialize database schema

CREATE TABLE IF NOT EXISTS repositories (
                                            id SERIAL PRIMARY KEY,
                                            owner TEXT NOT NULL,
                                            name TEXT NOT NULL,
                                            description TEXT,
                                            url TEXT,
                                            language TEXT,
                                            forks_count INT,
                                            stars_count INT,
                                            open_issues_count INT,
                                            watchers_count INT,
                                            created_at TIMESTAMP,
                                            updated_at TIMESTAMP,
                                            UNIQUE(owner, name)
    );

CREATE TABLE IF NOT EXISTS commits (
                                       id SERIAL PRIMARY KEY,
                                       sha TEXT UNIQUE NOT NULL,
                                       repository_id INT REFERENCES repositories(id) ON DELETE CASCADE,
    message TEXT,
    author_name TEXT,
    date TIMESTAMP,
    url TEXT
    );