-- name: CreateCatTable
CREATE TABLE Cat (
    id SERIAL,
    name VARCHAR(150),
    color VARCHAR(50),

    PRIMARY KEY (id)
)


-- name: CreatePsychoCat
INSERT INTO Cat (name, color) VALUES ('Puca', 'Orange');


-- name: CreateNormalCat
INSERT INTO Cat (name, color) VALUES (:name, :color);


-- name: UpdateColorById
UPDATE Cat
   SET color = :color
 WHERE id = :id;
