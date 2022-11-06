-- query: FindUserById
SELECT first_name,
       last_name,
       dob,
       email
  FROM user
 WHERE id = 1;


-- query: UpdateFirstNameById
UPDATE user
   SET first_name = 'Ernesto'
 WHERE id = 200;


-- query: DeleteUserById
DELETE FROM user
      WHERE id = $1;
