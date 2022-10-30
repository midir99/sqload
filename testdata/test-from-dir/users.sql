-- name: FindUserById
SELECT first_name,
       last_name,
       dob,
       email
  FROM user
 WHERE id = 1;


-- name: UpdateFirstNameById
UPDATE user
   SET first_name = 'Ernesto'
 WHERE id = 200;


-- name: DeleteUserById
DELETE FROM user
      WHERE id = $1;
