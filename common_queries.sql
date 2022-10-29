-- name: find_all_users
-- Find all users
SELECT *
  FROM uuser;


-- name: find_user_by_username
-- Find a user with the given username field
SELECT *
  FROM user
 WHERE username = 'neto';


-- name: find_user_by_id
-- Find a user with the given uuser_id
SELECT *
  FROM uuser
 WHERE uuser_id = %(uuser_id)s;


-- name: get_courses
  SELECT unitid,
         year,
         semester,
         unitcode,
         descr,
         estenrol,
         homepage,
         area,
         start_date
    FROM unit_view
ORDER BY year, semester, unitcode
   LIMIT %(limit)s
  OFFSET %(offset)s;
