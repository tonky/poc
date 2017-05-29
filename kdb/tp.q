\l qsql.q

/ use this to delete older tags from db on each update
/ .z.ts: {.P.prune_tags 10 ? .P.tags; .P.tp_upsert}

/ persist buffered table to disk
.z.ts: .P.tp_upsert

\t 1000
