/ use namespace .P for all defined functions

/ //////////////// tickerplant functions //////////////

.P.gen_tl: {([] tag:`symbol$(); ts:`s#`long$(); val:())}

.P.interval:{`long$(y - x) % 100}

/ find last value for a tag in a table, split in 100 buckets by time.
.P.gen_ts_int:{[s;e] i:.P.interval[s;e]; s + i* (1 + til 100)}
.P.join_on:{[tag;s;e] ([] tag:`sym$tag; ts:.P.gen_ts_int[s;e])}
.P.downsample_aj:{[tbl;tag;s;e] aj[`tag`ts;.P.join_on[tag;s;e];tbl]}


/ save partitioned tag to a separate db
.P.extr:{[tbl;tg] select from tbl where tag=`sym$tg}
.P.path:{`$raze ":/tmp/db/", string(`int$`sym$x), "/t/"}
.P.save_tag:{[tbl;tg] .P.path[tg] upsert .P.extr[tbl;tg]}

/ save all records with tags to respective dbs
.P.upsert_all:{tenum: .Q.en[`:/tmp/db/] x; .P.save_tag[tenum] peach distinct tenum[`tag]}

/ tickerplant persist to db function
.P.tp_upsert: {.tmp.upd: .tmp.t; .tmp.t: .P.gen_tl[]; .P.upsert_all .tmp.upd; delete upd from `.tmp}
.P.tp_add:{show count x; `.tmp.t upsert x}

/ initial empty column list for updates
.tmp.t: .P.gen_tl[]



/ //////////////// hdb functions //////////////

/ hdb reload db and update syms for client queries
.P.reload_hdb: {system"l ", "/tmp/db/"}

/ downsample with implicit reload
.P.downsample_tag:{[tag;s;e] .P.reload_hdb{}; .P.downsample_aj[select from t where int=`int$`sym?tag; tag; s; e]}




/ //////////////// utility and practive functions, for interactive testing //////////////
/ generate 'amt' symbols for tags distribution
/ .P.gen_syms:{[amt] (`$"t" ,/: string til amt)}

.P.tags: (`$"t" ,/: string til 10000)

/ empty table and row definition
.P.gen_t:{([] tag:`symbol$(); ts:`s#`timestamp$(); val:())}

/ generate enumerated tags
/ .P.gen_tags:{[amt;tags] `sym$amt?tags}

/ create 'amt' records for a tag, distributed randomly for 24h, starting with 'end'-24h, usually a .z.P - local timestamp
.P.gen_ts_24h:{[amt;end] asc (end-24:00:00) + amt?`timespan$24:00:00}

/ create 'amt' floating points
.P.gen_vals:{x?10.0}

/ generate 'historical'(24h worth) of amt records till 'end' - 24h
/ .P.gen_recs_24h: {[amt;end;tags] ([] tag:amt?.P.tags; ts:.P.gen_ts_24h[amt;end]; val:.P.gen_vals[amt])}

/ generate amt records starting _now_
.P.gen_row:{[amt;tags] ([] tag:amt?tags; ts:`long$(.z.p - `long$1970.01.01D00:00:00.000000) + til amt; val:{10?1.0} each til amt)}

.P.gen_recs: {[amt;tags] batch_size:amt&1000; .tmp.gen: .P.gen_tl[]; do[amt div batch_size; `.tmp.gen upsert .P.gen_row[batch_size;tags]]; .tmp.gen}

/ ignore tag
.P.join_on_notag:{[s;e] ([] ts:.P.gen_ts_int[s;e])}
.P.downsample_aj_notag:{[tbl;s;e] aj[`ts;.P.join_on_notag[s;e];tbl]}

/ find last value in 100 equal time buckets using xbar, too slow atm
/ .P.downsample:{[tbl;s;e] 1_ select by .P.interval[s;e] xbar ts from tbl where ts>s,ts<=e}

/ downsample last 24h
/ .P.ds24:{[tbl;tg] .P.downsample_aj_notag[select from tbl where int=`int$`sym$tg; .z.P-24:00:00; .z.P]}

.P.write_batch_fpath:{.P.upsert_all gen_tl[] upsert value "c"$read1`$":", x}

/ experimental pruning of records, older than 24h
.P.fpath:{"/tmp/db/", string(`int$`sym$x)}
.P.recent_recs: {select tag, ts, val from t where int=`int$`sym$x, ts>(`long$.z.p - `long$1970.01.02D00:00:00.000000)}
.P.prune_tag: {rr: .P.recent_recs[x]; system"rm -rf ", .P.fpath[x]; .P.upsert_all rr}
.P.prune_tags: {.P.prune_tag each x}
