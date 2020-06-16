--
-- split function
--    s   : string to split
--    del : delimiter
--    i   : index requested
--

DROP FUNCTION IF EXISTS SPLIT_STRING;

DELIMITER $

CREATE FUNCTION
   SPLIT_STRING ( s VARCHAR(1024) , del CHAR(1) , i INT)
   RETURNS VARCHAR(1024)
   DETERMINISTIC -- always returns same results for same input parameters
    BEGIN

        DECLARE n INT ;

        -- get max number of items
        SET n = LENGTH(s) - LENGTH(REPLACE(s, del, '')) + 1;

        IF i > n THEN
            RETURN NULL ;
        ELSE
            RETURN SUBSTRING_INDEX(SUBSTRING_INDEX(s, del, i) , del , -1 ) ;
        END IF;

    END
$

DELIMITER ;

UPDATE azure_pke_clusters set http_proxy = JSON_SET(http_proxy,
'$.http.scheme', SPLIT_STRING(JSON_UNQUOTE(http_proxy->'$.http.url'), ':', 1),
'$.http.host', SPLIT_STRING(SUBSTRING_INDEX(SUBSTRING_INDEX(SUBSTRING_INDEX(SUBSTRING_INDEX(JSON_UNQUOTE(http_proxy->'$.http.url'), '/', 3), '://', -1), '/', 1), '?', 1), ":", 1),
'$.http.port', CAST(IFNULL(SPLIT_STRING(JSON_UNQUOTE(http_proxy->'$.http.url'), ':', 3), 0) AS UNSIGNED),

'$.https.scheme', SPLIT_STRING(JSON_UNQUOTE(http_proxy->'$.https.url'), ':', 1),
'$.https.host', SPLIT_STRING(SUBSTRING_INDEX(SUBSTRING_INDEX(SUBSTRING_INDEX(SUBSTRING_INDEX(JSON_UNQUOTE(http_proxy->'$.https.url'), '/', 3), '://', -1), '/', 1), '?', 1), ":", 1),
'$.https.port', CAST(IFNULL(SPLIT_STRING(JSON_UNQUOTE(http_proxy->'$.https.url'), ':', 3), 0) AS UNSIGNED)
);

UPDATE azure_pke_clusters SET http_proxy = JSON_REMOVE(http_proxy, "$.https.url", "$.http.url");
