-- +goose Up
CREATE VIRTUAL TABLE repositories_search USING fts5(
    name,
    owner_name,
    description,
    content = 'repositories',
    content_rowid = 'id',
    tokenize = 'trigram'
);

-- +goose StatementBegin
CREATE TRIGGER repositories_search_ai AFTER INSERT ON repositories BEGIN
    INSERT INTO repositories_search(rowid, name, owner_name, description)
    VALUES (new.id, new.name, new.owner_name, new.description);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER repositories_search_ad AFTER DELETE ON repositories BEGIN
    INSERT INTO repositories_search(repositories_search, rowid, name, owner_name, description)
    VALUES ('delete', old.id, old.name, old.owner_name, old.description);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER repositories_search_au AFTER UPDATE OF name, owner_name, description ON repositories BEGIN
    INSERT INTO repositories_search(repositories_search, rowid, name, owner_name, description)
    VALUES ('delete', old.id, old.name, old.owner_name, old.description);
    INSERT INTO repositories_search(rowid, name, owner_name, description)
    VALUES (new.id, new.name, new.owner_name, new.description);
END;
-- +goose StatementEnd

INSERT INTO repositories_search(repositories_search) VALUES ('rebuild');

-- +goose Down
DROP TRIGGER IF EXISTS repositories_search_au;
DROP TRIGGER IF EXISTS repositories_search_ad;
DROP TRIGGER IF EXISTS repositories_search_ai;
DROP TABLE IF EXISTS repositories_search;
