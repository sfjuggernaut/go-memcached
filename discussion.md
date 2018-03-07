# Performance

### Limits

There is only one goroutine to accept new client connections. If new connections are continually coming and leaving, this could be a bottleneck. Persistent connections are better.

Each TCP connection does have some memory overhead. Take this into consideration when planning number of connections to the server as well as how much memory is dedicated to the server.


### Simple workload test

The short: in my **very** **simple** scenario I found my solution to handle roughly ~2/3 the queries per second (QPS) of stock memcached server: ~200k QPS vs ~310k QPS.

This was on a google cloud compute node with 8 cores and 30GB of memory on Debian `4.9.30`.

Note: both the workload generator and the server were run on the same compute node with 1/2 the cores dedicated to the workloader generator and 1/2 to the memcached server.


The workload generator used was [multirate](https://github.com/leverich/mutilate) and run with:
`./mutilate -s localhost -T 4 -c 8 -t 60`

The stock [memcached](https://github.com/memcached/memcached/wiki) was run with (16GB memory, max connections 1024, 4 threads):
`memcached -m 192384 -c 1024 -t 4`

My solution was run with (16GB memory, 32 hash buckets, 32 connection workers):
`~/go/bin/go-memcached -capacity 17179869184 -num-buckets 32 -num-workers 32`

The working set size of the cache being actually used seemed to never cross more than ~300MB, so max capacity and heavy eviction were never at issue.

A more realistic scenario would have a larger machine with all cores dedicated to memcached (either solution) with many separate client VMs. I would suspect in this scenario, the difference between stock memcached server and my solution to be even larger.

I expect the bottlenecks of my solution to be 1) GC and 2) locking of the hash table and eviction table.

For 1) GC, some of that is the nature of using Go over something like C/C++ and some of that is non-refined input parsing code.

For 2) locking, see notes below.

# Potential Improvements

Areas to examine and potentially improve is actually everything :) If it is not measured, then one should not assume it will perform well.

The major pieces to measure are: consuming incoming connections, parsing requests, caching, LRU, and stats gathering.

### Consuming connections

The buffered channel with workers to process incoming connections should scale reasonably well. goroutines are fairly lightweight. The `num-workers` param allows one to control how many workers are created (each being a separate goroutine). Each worker is responsible for handling a single client connection at a time.

### Parsing requests

There are likely some inefficiences with regards to allocation of memory to validate and parse the incoming requests (see `connReader()` and `parseRequest()`). In reality, we should just have to copy the data once (from the network connection buffer to memcached's buffers). It is also  worthwhile to examine using the binary protocol over the text protocol.

As an example, each worker could up-front allocate a `Request` struct and re-use that object instead of re-creating a new one for each request the client issues. This is somewhat dependent on the workload.

### Caching and LRU


The current cache and LRU implementation is pretty straight forward - an array of buckets, each bucket with a hash map for storing (key, value) tuples and an evict list implemented as a doubly linked list . A lock on the bucket is required to update the evict list. This means that locking (RW) is necessary on all types of requests for the entire bucket - even Get calls (ie: reads).

A simple thing to try, replace all bucket locking with a per-bucket "EventLoop" goroutine and channel. Gets and updates to the hash map would use a call and response pattern via the channel. This pattern would be more idiomatic Go (the whole share by communicating over sharing memory idea).

We could look into improving Get performance by potentially separating the retrieval of data from the hash map from maintaining the evict list. This could be achieved through Go channels and a "EvictLoop" goroutine. This would requie implementing our own doubly linked list instead of relying on the stock [list pkg](https://golang.org/pkg/container/list/), as you cannot create an element for the list w/out also inserting it at the same time. So you cannot have the SET operation add to the evict list *after* its added to the hash map in the current design.

A longer term solution might be to examine and implement the slab allocator that stock memcached uses. Note: no memory is pre-allocated in my current solution.

Lastly, we should look at the distribution of entries to buckets and explore other hashing techniques. Current solution uses Go's built-in FNV-1a hashing algorithm, but others (such as murmur3) should be explored.

### Stats

Perhaps not a problem, but we should ensure that storing and retrieving of stats doesn't induce performance hits.


### More rigorous testing

I've done a decent job at testing, but more heavy duty testing is necessary to consider the solution stable and reliable. Also code reviews.

# Monitoring and Management

### Monitoring and Alerting

A json endpoint (`/stats`) is exposed over the admin HTTP interface. This endpoint returns the current stats of the live running process. More stats can be easily added via `stats.go`.

It should be easy to have stats consumers (such as data dog, in-house solution, etc.) pull from this endpoint to populate graphs / dashboards.

Alerting can then be built on top of the graphs / dashboards.

### Management
The available params to adjust are:
- port : port to run memcached server
- admin-http-port : port to run admin HTTP server (for stats and profiling)
- capacity : maximum number of bytes to store (memory limit of server)
- num-workers : number of workers to process incoming connections
- max-num-connections: maximum number of simultaneous connections (clients block while at this limit)
- num-buckets : number of buckets in the hash table of the cache

It should be easy to build and run this code as a binary and manage via something like `runit`.

### Profiling

Use [pprof](https://golang.org/pkg/net/http/pprof/) endpoints to examine CPU usage and heap allocations. See `adminHttpServerStart()` for available routes.

Use [perf](https://perf.wiki.kernel.org/index.php/Tutorial) to see problems such as lock contention via `sudo perf top -p <pid>`.

Fun visual aids such as [torch](https://github.com/uber/go-torch) and [flamegraphs](https://github.com/brendangregg/FlameGraph) can also be useful for understanding how the system works.

Inspect and make better!


# Testing

### Unit
There are unit tests in `server_test.go` and `cache_test.go`.

### Manual
I did some simple manually testing with `telnet` and bradfitz's [client](github.com/bradfitz/gomemcache/memcache) to give some faith of compatability.

### Stress
For more instense testing, I ran workload generators [mutilate](https://github.com/leverich/mutilate) and [twemperf](https://github.com/twitter/twemperf) to not only obtain perf numbers but give some reasonable confidence that the implemenation is stable.

### Future
More testing should be done with different workloads (instead of the main one I used via `mutilate`). A variance on R vs W %, different data block sizes (small, large, mixed), large number of clients, ensuring heavy eviction would be good examples.

No testing was done outside standard ASCII values.
