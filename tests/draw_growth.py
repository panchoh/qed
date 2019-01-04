#!/usr/bin/env python 

from plotly import tools
import plotly.plotly as py 
import plotly.graph_objs as go
from plotly.offline import download_plotlyjs, init_notebook_mode, plot, iplot
import sys
import json

def group_chunks(serie, chunk_size):
    return [serie[i * chunk_size:(i+1) * chunk_size] for i in range((len(serie) + chunk_size - 1) // chunk_size)]

def mean_growth(serie, chunk_size): 
    serie_growth = [ ((y - x) / x) * 100 for x, y in zip(serie, serie[1:])]
    serie_growth_chunked = group_chunks(serie_growth, chunk_size)
    return [ reduce(lambda x, y: x+y, chunk) / len(chunk) for chunk in serie_growth_chunked]

def main(argv):
    inputfile = sys.argv[1]
    with open(inputfile) as f:
        lines = f.readlines()
        metrics = [json.loads(line) for line in lines] 

        minutes = [i for i, x in enumerate(metrics, 1)]
        mutations = [m['badger.mutate']['99%'] for m in metrics]
        ranges =  [m['badger.get_range']['99%'] for m in metrics]
        prunings_after = [m['hyper.pruning.after_cache']['99%'] for m in metrics]
        prunings_leaves = [m['hyper.pruning.get_leaves']['99%'] for m in metrics]
        visits = [m['hyper.visiting']['99%'] for m in metrics]
        prunings_before = [ x - y - z  for x, y, z in zip(visits, prunings_after, prunings_leaves)]
        adds = [m['hyper.add']['99%'] for m in metrics]
        test_adds = [m['hyper.test_add']['99%'] for m in metrics]


        minutes_chunked = group_chunks(minutes, 10)
        minutes_reduced = [i for i in range(len(minutes_chunked))]

        mutations_growth = mean_growth(mutations, 10)
        prunings_before_growth = mean_growth(prunings_before, 10)
        prunings_leaves_growth = mean_growth(prunings_leaves, 10)
        prunings_after_growth = mean_growth(prunings_after, 10)
        visits_growth = mean_growth(visits, 10)

        traceMutations = go.Bar(
            x=minutes_reduced, 
            y=mutations_growth,
            name='Mutations'
        )
        traceBefore = go.Bar(
            x=minutes_reduced, 
            y=prunings_before_growth,
            name='Before'
        )
        traceLeaves = go.Bar(
            x=minutes_reduced,
            y=prunings_leaves_growth,
            name="Get leaves"
        )
        traceAfter = go.Bar(
            x=minutes_reduced,
            y=prunings_after_growth,
            name='After'
        )
        traceVisiting = go.Bar(
            x=minutes_reduced,
            y=visits_growth,
            name='Visits'
        )

        fig = tools.make_subplots(rows=3, cols=2, specs=[[{},{}],[{},{}], [{},{}]],
                                subplot_titles=('Average growth rate mutations','Average growth rate before get leaves',
                                'Average growth get leaves', 'Average growth after get leaves',
                                'Average growth visiting'))

        fig.append_trace(traceMutations, 1, 1)
        fig.append_trace(traceBefore, 1, 2)
        fig.append_trace(traceLeaves, 2, 1)
        fig.append_trace(traceAfter, 2, 2)
        fig.append_trace(traceVisiting, 3, 1)

        fig.layout['yaxis1'].update(range=[-5, 20])
        fig.layout['yaxis2'].update(range=[-5, 20])
        fig.layout['yaxis3'].update(range=[-5, 20])
        fig.layout['yaxis4'].update(range=[-5, 20])
        fig.layout['yaxis5'].update(range=[-5, 20])

        plot(fig)



if __name__ == "__main__":
    main(sys.argv)