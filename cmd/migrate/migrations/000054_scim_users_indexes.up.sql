-- Add indexes for SCIM users pagination queries to improve performance
-- and prevent timeouts on large datasets

-- Index for ListSCIMUsers query: WHERE scim_directory_id = $1 AND id >= $2 ORDER BY id
CREATE INDEX IF NOT EXISTS scim_users_scim_directory_id_id
    ON scim_users(scim_directory_id, id);

-- Index for ListSCIMUsersInSCIMGroup query with EXISTS subquery
-- Note: There's already a unique index on (scim_user_id, scim_group_id), but this
-- index on (scim_group_id, scim_user_id) is more optimal for queries filtering by
-- scim_group_id first (which is the pattern in the EXISTS subquery)
CREATE INDEX IF NOT EXISTS scim_user_group_memberships_scim_group_id_scim_user_id
    ON scim_user_group_memberships(scim_group_id, scim_user_id);
