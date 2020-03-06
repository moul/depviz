export const computeLayoutConfig = (layout) => {
  let layoutConfig;

  switch (layout) {
    case "circle":
      layoutConfig = {
        name: 'circle',
        avoidOverlap: true,
      }
      break;
    case "cose":
      layoutConfig = {
        name: 'cose',
        animate: false,
        componentSpacing: 0.5,
        nodeOverlap: 2,
        nodeRepulsion: 0.5,
        nestingFactor: 19,
        gravity: 200,
        numIter: 2000,
        coolingFactor: 0.2,
      }
      break;
    case "breadthfirst":
      layoutConfig = {
        name: 'breadthfirst',
      }
      break;
    case "concentric":
      layoutConfig = {
        name: 'concentric',
      }
      break;
    case "grid":
      layoutConfig = {
        name: 'grid',
        condense: true,
      }
      break;
    case "random":
      layoutConfig = {
        name: 'random',
      }
      break;
    case "cola":
      layoutConfig = {
        name: 'cola',
        animate: false,
        refresh: 1,
        padding: 30,
        maxSimulationTime: 100,
      }
      break;
    case "elk":
      layoutConfig = {
        name: 'elk',
        elk: {
          'zoomToFit': true,
          'algorithm': 'mrtree',
          'separateConnectedComponents': false
        },
      }
      break;
    case "gantt":
      layoutConfig = {
        name: 'gantt',
      }
      break;
    case "flow":
      layoutConfig = {
        name: 'flow',
      }
      break;
    default:
      break;
  }

  return layoutConfig;
}
