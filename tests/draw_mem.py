#!/usr/bin/env python 

from plotly import tools
import plotly.plotly as py 
import plotly.graph_objs as go
from plotly.offline import download_plotlyjs, init_notebook_mode, plot, iplot
import sys
import json

def main(argv):
    inputfile = argv[1]
    with open(inputfile) as f:
        lines = f.readlines()
        metrics = [json.loads(line) for line in lines] 

        minutes = [i for i, x in enumerate(metrics, 1)]
        rate1m = [m['hyper.test_add']['1m.rate'] for m in metrics]

        totalAlloc = [m['runtime.MemStats.TotalAlloc']['value'] for m in metrics] # TotalAlloc is cumulative bytes allocated for heap objects.
        lookups = [m['runtime.MemStats.Lookups']['value'] for m in metrics] # Lookups is the number of pointer lookups performed by the runtime.
        sys = [m['runtime.MemStats.Sys']['value'] for m in metrics] # Sys is the total bytes of memory obtained from the OS.
        mallocs = [m['runtime.MemStats.Mallocs']['value'] for m in metrics] # Mallocs is the cumulative count of heap objects allocated.
        frees = [m['runtime.MemStats.Frees']['value'] for m in metrics] #  Frees is the cumulative count of heap objects freed.
        heapAlloc = [m['runtime.MemStats.HeapAlloc']['value'] for m in metrics] # Allocated heap objects include all reachable objects.
        heapSys = [m['runtime.MemStats.HeapSys']['value'] for m in metrics] # HeapSys is bytes of heap memory obtained from the OS.
        heapIdle = [m['runtime.MemStats.HeapIdle']['value'] for m in metrics] # HeapIdle is bytes in idle (unused) spans.
        heapInuse = [m['runtime.MemStats.HeapInuse']['value'] for m in metrics] # HeapInuse is bytes in in-use spans.
        heapReleased = [m['runtime.MemStats.HeapReleased']['value'] for m in metrics] # HeapReleased is bytes of physical memory returned to the OS.
        heapObjects = [m['runtime.MemStats.HeapObjects']['value'] for m in metrics] # HeapObjects is the number of allocated heap objects.
        stackInuse = [m['runtime.MemStats.StackInuse']['value'] for m in metrics] # StackInuse is bytes in stack spans.
        stackSys = [m['runtime.MemStats.StackInuse']['value'] for m in metrics] # StackSys is bytes of stack memory obtained from the OS.
        mSpanInuse = [m['runtime.MemStats.MSpanInuse']['value'] for m in metrics] # MSpanInuse is bytes of allocated mspan structures.
        mSpanSys = [m['runtime.MemStats.MSpanSys']['value'] for m in metrics] # MSpanSys is bytes of memory obtained from the OS for mspan structures.
        mCacheInuse = [m['runtime.MemStats.MCacheInuse']['value'] for m in metrics] # MCacheInuse is bytes of allocated mcache structures.
        mCacheSys = [m['runtime.MemStats.MCacheSys']['value'] for m in metrics] # MCacheSys is bytes of memory obtained from the OS for mcache structures.
        buckHashSys = [m['runtime.MemStats.BuckHashSys']['value'] for m in metrics] # BuckHashSys is bytes of memory in profiling bucket hash tables.
        nextGC = [m['runtime.MemStats.NextGC']['value'] for m in metrics] # NextGC is the target heap size of the next GC cycle.
        lastGC = [m['runtime.MemStats.LastGC']['value'] for m in metrics] # LastGC is the time the last garbage collection finished, as nanoseconds since 1970 (the UNIX epoch).
        pauseTotalNs = [m['runtime.MemStats.PauseTotalNs']['value'] for m in metrics] # PauseTotalNs is the cumulative nanoseconds in GC stop-the-world pauses since the program started.
        pauseTotalNs_deltas = [max(0, y - x) / 1000000 for x, y in zip(pauseTotalNs, pauseTotalNs[1:])]
        pauseNs = [m['runtime.MemStats.PauseNs']['95%'] for m in metrics] # PauseNs is a circular buffer of recent GC stop-the-world pause times in nanoseconds.
        numGC = [0] + [m['runtime.MemStats.NumGC']['value'] for m in metrics] # NumGC is the number of completed GC cycles.
        numGC_deltas = [max(0, y - x) for x, y in zip(numGC, numGC[1:])]
        gcCpuFraction = [m['runtime.MemStats.GCCPUFraction']['value'] for m in metrics] # GCCPUFraction is the fraction of this program's available CPU time used by the GC since the program started.
        numGoroutine = [m['runtime.NumGoroutine']['value'] for m in metrics] # NumGoroutine is the number of goroutines that currently exist.
        numThread = [m['runtime.NumThread']['value'] for m in metrics] # NumThread is the number of threads that currently exist.

        cacheSize = [m['cache.size']['value'] for m in metrics]
        cacheGets = [m['cache.gets']['count'] for m in metrics]
        cachePuts = [m['cache.puts']['count'] for m in metrics]
        total_adds = [m['hyper.add']['count'] for m in metrics]

        cachePutsRatio = [ x/y for x, y in zip(cachePuts, total_adds)]
        cacheGetsRatio = [ x/y for x, y in zip(cacheGets, total_adds)]



        traceRate1m = go.Scatter(
            x=minutes,
            y=rate1m,
            name='Rate 1m'
        )
        
        traceHeapObjects = go.Scatter(
            x=minutes,
            y=heapObjects,
            name='Heap Objects'
        )

        traceNextGC = go.Scatter(
            x=minutes,
            y=nextGC,
            name='Next GC size'
        )
        traceHeapAlloc = go.Scatter(
            x=minutes,
            y=heapAlloc,
            name='Heap Alloc (bytes)'
        )
        traceHeapInuse = go.Scatter(
            x=minutes,
            y=heapInuse,
            name='Heap in use (bytes)'
        )
        traceHeapIdle = go.Scatter(
            x=minutes,
            y=heapIdle,
            name='Heap idle (bytes)'
        )
        traceHeapReleased = go.Scatter(
            x=minutes,
            y=heapReleased,
            name='Heap Released (bytes)'
        )

        tracePauseNs = go.Scatter(
            x=minutes,
            y=pauseTotalNs_deltas,
            name='GC Pause (ms)'
        )

        traceNumGoroutines = go.Scatter(
            x=minutes,
            y=numGoroutine,
            name='Num Goroutines'
        )
        traceNumThread = go.Scatter(
            x=minutes,
            y=numThread,
            name='Num threads'
        )

        traceLookups = go.Scatter(
            x=minutes,
            y=lookups,
            name='Pointer lookups'
        )

        traceSys = go.Scatter(
            x=minutes,
            y=sys,
            name='Sys (bytes)'
        )
        traceHeapSys = go.Scatter(
            x=minutes,
            y=heapSys,
            name='Heap Sys (bytes)'
        )
        traceStackSys = go.Scatter(
            x=minutes,
            y=stackSys,
            name='Stack Sys (bytes)'
        )
        traceMSpanSys = go.Scatter(
            x=minutes,
            y=mSpanSys,
            name='Span Sys (bytes)'
        )
        traceBuckHashSys = go.Scatter(
            x=minutes,
            y=buckHashSys,
            name='BuckHash Sys (bytes)'
        )
        traceMCacheSys = go.Scatter(
            x=minutes,
            y=mCacheSys,
            name='MCache Sys (bytes)'
        )

        traceStackInuse = go.Scatter(
            x=minutes,
            y=stackInuse,
            name='Stack in use (bytes)'
        )
        traceMSpanInuse = go.Scatter(
            x=minutes,
            y=mSpanInuse, 
            name='MSpan in use (bytes)'
        )
        traceMCacheInuse = go.Scatter(
            x=minutes,
            y=mCacheInuse,
            name='MCache in use (bytes)'
        )

        traceNumGC = go.Scatter(
            x=minutes,
            y=numGC_deltas,
            name='Num GC cycles'
        )

        traceCacheSize = go.Scatter(
            x=minutes,
            y=cacheSize,
            name='Cache size (elems)'
        )

        traceCachePutsRatio = go.Scatter(
            x=minutes,
            y=cachePutsRatio,
            name='Puts per add'
        )
        traceCacheGetsRatio = go.Scatter(
            x=minutes,
            y=cacheGetsRatio,
            name='Gets per add'
        )

        fig = tools.make_subplots(rows=3, cols=3, specs=[[{}, {}, {}], [{}, {}, {}], [{}, {}, {}]],
                                subplot_titles=(
                                    'Rates', 'Memory obtained from the OS', 'Number of threads & goroutines',
                                    'Heap use', 'Other mem use', 'GC cycles',
                                    'Heap allocs', 'GC Pause time', 'Cache num elements', 
                                ))

        fig.append_trace(traceRate1m, 1, 1)

        fig.append_trace(traceSys, 1, 2)
        fig.append_trace(traceHeapSys, 1, 2)
        fig.append_trace(traceStackSys, 1, 2)
        fig.append_trace(traceMSpanSys, 1, 2)
        fig.append_trace(traceBuckHashSys, 1, 2)
        fig.append_trace(traceMCacheSys, 1, 2)

        fig.append_trace(traceNumGoroutines, 1, 3)
        fig.append_trace(traceNumThread, 1, 3)

        fig.append_trace(traceHeapAlloc, 2, 1)
        fig.append_trace(traceHeapInuse, 2, 1)
        fig.append_trace(traceHeapIdle, 2, 1)
        fig.append_trace(traceHeapReleased, 2, 1)
        fig.append_trace(traceNextGC, 2, 1)

        fig.append_trace(traceStackInuse, 2, 2)
        fig.append_trace(traceMSpanInuse, 2, 2)
        fig.append_trace(traceMCacheInuse, 2, 2)

        fig.append_trace(traceNumGC, 2, 3)

        fig.append_trace(traceHeapObjects, 3, 1)

        fig.append_trace(tracePauseNs, 3, 2)

        fig.append_trace(traceCachePutsRatio, 3, 3)
        fig.append_trace(traceCacheGetsRatio, 3, 3)

        plot(fig)

if __name__ == "__main__":
    main(sys.argv)
