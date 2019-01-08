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

        rate1m = [m['hyper.test_add']['1m.rate'] for m in metrics]
        rate5m = [m['hyper.test_add']['5m.rate'] for m in metrics]
        rate15m = [m['hyper.test_add']['15m.rate'] for m in metrics]
        events = [m['hyper.test_add']['count'] for m in metrics]

        traceRate1m = go.Scatter(
            x=minutes, 
            y=rate1m,
            name='1m Rate'
        )
        traceRate5m = go.Scatter(
            x=minutes, 
            y=rate5m,
            name='5m Rate'
        )
        traceRate15m = go.Scatter(
            x=minutes, 
            y=rate15m,
            name='15m Rate'
        ) 
        traceEvents = go.Scatter(
            x=minutes,
            y=events,
            name="Num events"
        )  

        fig = tools.make_subplots(rows=2, cols=1, specs=[[{}],[{}]],
                                subplot_titles=('Rates', 'Num events'))

        fig.append_trace(traceRate1m, 1, 1)
        fig.append_trace(traceRate5m, 1, 1)
        fig.append_trace(traceRate15m, 1, 1)
        fig.append_trace(traceEvents, 2, 1)

        fig.layout['yaxis1'].update(range=[0, 8000])
        
        plot(fig)

if __name__ == "__main__":
    main(sys.argv)