-- Remove indexes added for SCIM users pagination queries

DROP INDEX IF EXISTS scim_users_scim_directory_id_id;
DROP INDEX IF EXISTS scim_user_group_memberships_scim_group_id_scim_user_id;
