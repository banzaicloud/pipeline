UPDATE azure_pke_clusters SET http_proxy = JSON_SET(http_proxy,

'$.https.url',
IFNULL(CONCAT(
IFNULL(JSON_UNQUOTE(http_proxy->'$.https.scheme'), 'https'),
"://",
JSON_UNQUOTE(http_proxy->'$.https.host'),
':',
JSON_UNQUOTE(http_proxy->'$.https.port')
),''),

'$.http.url',
IFNULL(CONCAT(
IFNULL(JSON_UNQUOTE(http_proxy->'$.http.scheme'), 'http'),
"://",
JSON_UNQUOTE(http_proxy->'$.http.host'),
':',
JSON_UNQUOTE(http_proxy->'$.http.port')
), ''));

UPDATE azure_pke_clusters SET http_proxy = JSON_REMOVE(http_proxy, '$.https.scheme', '$.https.host', '$.https.port', '$.http.scheme', '$.http.host', '$.http.port');
