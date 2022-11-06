-- query: CreateCatTable
CREATE TABLE Cat (
    id SERIAL,
    name VARCHAR(150),
    color VARCHAR(50),

    PRIMARY KEY (id)
);


-- query: CreatePsychoCat
INSERT INTO Cat (name, color) VALUES ('Puca', 'Orange');


-- query: CreateNormalCat
INSERT INTO Cat (name, color) VALUES (:name, :color);


-- query: UpdateColorById
UPDATE Cat
   SET color = :color
 WHERE id = :id;
