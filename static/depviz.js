$(document).ready(function() {
  $('#targets').focus();

  let searchParams = new URLSearchParams(window.location.search)
  if (searchParams.has("targets")) {
    let targets = searchParams.get("targets")
    $('#targets').val(targets)
    $("#result").html("loading JSON...")
    let url = "/api/graph?targets=" + searchParams.get("targets")
    let taskTemplate = $('#task-template').html()
    Mustache.parse(taskTemplate)

    $.ajax({
      url: url,
      success: function(result, status, xhr) {
        var nodes = [];
        var edges = [];

        result.tasks.forEach(task => {
          // node
          let svg = Mustache.render(taskTemplate, task)
          let svgUrl = "data:image/svg+xml;charset=utf-8,"+encodeURIComponent(svg);
          let node = {
            id: task.id,
            image: svgUrl,
            shape: 'image',
            multi: 'html',
          }
          nodes.push(node)
          //console.log(task);

          // relationships
          //if (task.has_owner !== undefined) { edges.push({from: task.has_owner, to: task.id}) }
          //if (task.has_author !== undefined) { edges.push({from: task.has_author, to: task.id}) }
          //if (task.has_milestone !== undefined) { edges.push({from: task.has_milestone, to: task.id}) }
          if (task.is_depending_on !== undefined) {
            task.is_depending_on.forEach(other => edges.push({
              from: task.id,
              to: other,
              relation: "is_depending_on",
              arrows: "to",
              color: {color: 'red'},
            }))
          }
          if (task.is_blocking !== undefined) {
            task.is_blocking.forEach(other => edges.push({
              from: other,
              to: task.id,
              relation: "is_depending_on",
              arrows: "to",
              color: {color: 'red'},
            })) }
          //if (task.has_assignee !== undefined) { task.has_assignee.forEach(other => edges.push({from: other, to: task.id})) }
          //if (task.has_reviewer !== undefined) { task.has_reviewer.forEach(other => edges.push({from: other, to: task.id})) }
          //if (task.has_label !== undefined) { task.has_label.forEach(other => edges.push({from: other, to: task.id})) }
          //if (task.is_related_with !== undefined) { task.is_related_with.forEach(other => edges.push({from: other, to: task.id})) }
          //if (task.is_part_of !== undefined) { task.is_part_of.forEach(other => edges.push({from: other, to: task.id})) }
          //if (task.has_part !== undefined) { task.has_part.forEach(other => edges.push({from: other, to: task.id})) }
        })
        // create a network
        var container = document.getElementById('network');
        var data = {
          nodes: nodes,
          edges: edges,
        };
        var options = {
          edges: {
            smooth: {
              forceDirection: 'none',
              roundness: 0.8,
            },
            length: 100,
          },
          nodes: {
            size: 24,
          },
          physics: {
            minVelocity: 0.1,
            //solver: 'repulsion',
            //solver: 'hierarchicalRepulsion',
            solver: 'barnesHut',
            barnesHut: {
              avoidOverlap: 0.5,
            },
            hierarchicalRepulsion: {
              avoidOverlap: 0.6,
            },
            stabilization: false,
          },
        };
        var network = new vis.Network(container, data, options);
        //console.log("network", network);
      },
      error: function(xhr, status, error) {
        console.error("failed", xhr, status, error);
        alert("failed: " +  error);
      },
    })
  }
});
