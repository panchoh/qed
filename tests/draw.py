#!/usr/bin/env python 

from plotly import tools
import plotly.plotly as py 
import plotly.graph_objs as go
from plotly.offline import download_plotlyjs, init_notebook_mode, plot, iplot
import sys
import json


def main(argv):
    inputfile = sys.argv[1]
    with open(inputfile) as f:
        lines = f.readlines()
        metrics = [json.loads(line) for line in lines] 
        
        minutes = [i for i, x in enumerate(metrics, 1)]
        mutations = [m['badger.mutate']['95%'] for m in metrics]
        ranges =  [m['badger.get_range']['95%'] for m in metrics]
        prunings = [m['hyper.pruning']['95%'] for m in metrics]
        prunings_after = [m['hyper.pruning.after_cache']['95%'] for m in metrics]
        prunings_leaves = [m['hyper.pruning.get_leaves']['95%'] for m in metrics]
        leaves = [m['hyper.pruning.leaves']['99%'] for m in metrics]
        visits = [m['hyper.visiting']['95%'] for m in metrics]
        prunings_before = [ x - y - z  for x, y, z in zip(visits, prunings_after, prunings_leaves)]
        adds = [m['hyper.add']['95%'] for m in metrics]
        test_adds = [m['hyper.test_add']['95%'] for m in metrics]
        rate1m = [m['hyper.test_add']['1m.rate'] for m in metrics]
        rate5m = [m['hyper.test_add']['5m.rate'] for m in metrics]
        rate15m = [m['hyper.test_add']['15m.rate'] for m in metrics]
        gets = [m['cache.gets']['95%'] for m in metrics]
        puts = [m['cache.puts']['95%'] for m in metrics]
        size = [m['cache.size']['value'] for m in metrics]
        events = [m['hyper.add']['count'] for m in metrics]


        #muts = [0] + [m['badger.mutate']['count'] for m in metrics]
        #muts_deltas = [ y - x for x, y in zip(muts, muts[1:])]

        store_blocked_puts_total = [0] + [m['store.blocked_puts_total']['value'] for m in metrics]
        store_blocked_puts_deltas = [ y - x for x, y in zip(store_blocked_puts_total, store_blocked_puts_total[1:])]

        store_disk_reads_total = [0] + [m['store.disk_reads_total']['value'] for m in metrics]
        store_disk_reads_deltas = [ y - x for x, y in zip(store_disk_reads_total, store_disk_reads_total[1:])]
        
        store_disk_writes_total = [0] + [m['store.disk_writes_total']['value'] for m in metrics]
        store_disk_writes_deltas = [ y - x for x, y in zip(store_disk_writes_total, store_disk_writes_total[1:])]

        store_gets_total = [0] + [m['store.gets_total']['value'] for m in metrics]
        store_gets_deltas = [ y - x for x, y in zip(store_gets_total, store_gets_total[1:])]

        store_memtable_gets_total = [0] + [m['store.memtable_gets_total']['value'] for m in metrics]
        store_memtable_gets_deltas = [ y - x for x, y in zip(store_memtable_gets_total, store_memtable_gets_total[1:])]

        store_puts_total = [0] + [m['store.puts_total']['value'] for m in metrics]

        store_puts_deltas = [ y - x for x, y in zip(store_puts_total, store_puts_total[1:])]
        
       # store_blocked_puts_total = [m['store.blocked_puts_total']['value'] for m in metrics]
       # store_blocked_puts_total = [m['store.blocked_puts_total']['value'] for m in metrics]

        trace1_1 = go.Scatter(
            x=minutes, 
            y=mutations,
            name='Mutation 95%'
        )
        trace1_2 = go.Scatter(
            x=minutes, 
            y=ranges,
            name='Get Range 95%'
        )
        trace1_3 = go.Scatter(
            x=minutes, 
            y=prunings,
            name='Pruning 95%'
        )
        trace1_4 = go.Scatter(
            x=minutes, 
            y=visits,
            name='Visiting 95%'
        )
        trace1_5 = go.Scatter(
            x=minutes, 
            y=adds,
            name='Add 95%'
        )
        trace1_6 = go.Scatter(
            x=minutes, 
            y=test_adds,
            name='Test Add 95%'
        )
        trace2_1 = go.Scatter(
            x=minutes, 
            y=rate1m,
            name='1m Rate'
        )
        trace2_2 = go.Scatter(
            x=minutes, 
            y=rate5m,
            name='5m Rate'
        )
        trace2_3 = go.Scatter(
            x=minutes, 
            y=rate15m,
            name='15m Rate'
        )                
        trace3_1 = go.Scatter(
            x=minutes, 
            y=gets,
            name='Cache gets 95%'
        )
        trace3_2 = go.Scatter(
            x=minutes, 
            y=puts,
            name='Cache puts 95%'
        ) 
        trace3_3 = go.Scatter(
            x=minutes, 
            y=size,
            name='Cache size'
        )          
        trace4_1 = go.Scatter(
            x=minutes, 
            y=store_blocked_puts_deltas,
            name='Blocked puts'
        )
        trace5_1 = go.Scatter(
            x=minutes, 
            y=store_disk_reads_deltas,
            name='Disk reads'
        )
        trace5_2 = go.Scatter(
            x=minutes, 
            y=store_disk_writes_deltas,
            name='Disk writes'
        )
        trace4_2 = go.Scatter(
            x=minutes, 
            y=store_gets_deltas,
            name='Gets'
        )
        trace4_3 = go.Scatter(
            x=minutes, 
            y=store_memtable_gets_deltas,
            name='Memtable gets'
        )
        trace4_4 = go.Scatter(
            x=minutes, 
            y=store_puts_deltas,
            name='Puts'
        )

        traceEvents = go.Scatter(
            x=minutes,
            y=events,
            name='Num events'
        )

        tracePruningAfter = go.Scatter(
            x=minutes,
            y=prunings_after,
            name="After cache (95%)"
        )
        tracePruningLeaves = go.Scatter(
            x=minutes,
            y=prunings_leaves,
            name="Get leaves (95%)"
        )
        traceLeaves = go.Scatter(
            x=minutes,
            y=leaves,
            name="Num leaves (99%)"
        )
        tracePruningBefore = go.Scatter(
            x=minutes,
            y=prunings_before,
            name="On cache (95%)"
        )

        fig = tools.make_subplots(rows=3, cols=2, specs=[[{'colspan': 2}, None], [{}, {}], [{}, {}]],
                                subplot_titles=('Stack Times', 'Rates', 'Cache use time', 'Storage Use', 'Bytes R/W'))
        fig.append_trace(trace1_1, 1, 1)
        fig.append_trace(trace1_2, 1, 1)
        fig.append_trace(trace1_3, 1, 1)
        fig.append_trace(trace1_4, 1, 1)
        fig.append_trace(trace1_5, 1, 1)
        fig.append_trace(trace1_6, 1, 1)
        fig.append_trace(tracePruningAfter, 1, 1)
        fig.append_trace(tracePruningLeaves, 1, 1)
        fig.append_trace(tracePruningBefore, 1, 1)
        fig.append_trace(traceLeaves, 1, 1)
        fig.append_trace(trace2_1, 2, 1)
        fig.append_trace(trace2_2, 2, 1)
        fig.append_trace(trace2_3, 2, 1)
        fig.append_trace(trace3_1, 2, 2)
        fig.append_trace(trace3_2, 2, 2)
        fig.append_trace(trace3_3, 2, 2)
        fig.append_trace(traceEvents, 2, 2)
        fig.append_trace(trace4_1, 3, 1)
        fig.append_trace(trace4_2, 3, 1)
        fig.append_trace(trace4_3, 3, 1)
        fig.append_trace(trace4_4, 3, 1)
        fig.append_trace(trace5_1, 3, 2)
        fig.append_trace(trace5_2, 3, 2)

        fig.layout['yaxis2'].update(range=[0, 3000])
        
        #fig = go.Figure(data=data, layout=layout)
        plot(fig)

if __name__ == "__main__":
    main(sys.argv)
