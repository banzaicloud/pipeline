UPDATE organizations SET provider = "" WHERE name = "spotguides";
UPDATE organizations SET provider = "user" WHERE provider = "github" and created_at >= "2019-02-23 19:37:29";