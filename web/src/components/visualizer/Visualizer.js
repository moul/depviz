/* eslint-disable react/prop-types */

import React from "react";
import Card from "./cardTemplate";
import cytoscape from "cytoscape";
import nodeHtmlLabel from "cytoscape-node-html-label";
import { computeLayoutConfig } from "./utils";
import "./card.scss"

const Visualizer = ({ data, layout }) => {
  const { tasks } = data || {};
  let cy;
  let layoutConfig = computeLayoutConfig(layout);

  if (tasks) {
    let config = {
      container: document.getElementById('cy'),
      elements: [],
      style: [{
        selector: 'node.Issue, node.MergeRequest',
        style: {
          "overlay-padding": "5px",
          "overlay-opacity": 0,
          "width": "510px",
          "height": "260px",
          shape: 'rectangle',
          'background-color': 'white',
        },
      }, {
        selector: 'node:parent',
        style: {
          'background-color': 'lightblue',
          opacity: 0.5,
          label: 'data(local_id)',
          padding: 50,
        },
      }],
      layout: layoutConfig
    }

    let nodes = []
    let edges = []

    tasks.forEach(task => {
      let node = {
        data: task,
        classes: task.kind,
      }
      switch (task.state) {
      case "Open":
        switch (task.kind) {
        case "MergeRequest":
          node.data.card_classes = 'in-progress'
          break;
        default:
          node.data.card_classes = 'open'
          break;
        }
        break;
      case "Closed":
      case "Merged":
        node.data.card_classes = 'closed'
        break;
      default:
        console.warn('unsupported task.state', task)
        node.data.card_classes = 'error'
      }

      switch (task.kind) {
      case "Issue":
        node.data.bgcolor = 'lightblue'
        node.data.is_issue = true
        node.data.progress = 0.5
        node.data.card_classes += ' issue'
        break
      case "Milestone":
        node.data.bgcolor = 'lightgreen'
        node.data.is_milestone = true
        node.data.card_classes += ' milestone'
        break
      case "MergeRequest":
        node.data.bgcolor = 'purple'
        node.data.is_mergerequest = true
        node.data.card_classes += ' pr'
        break
      default:
        console.warn("unsupported task.kind", task)
        node.data.bgcolor = 'grey'
      }
      // common
      node.data.nb_parents = 0
      node.data.nb_children = 0
      node.data.nb_related = 0
      node.data.nb_parents += (task.is_blocking !== undefined ? task.is_blocking.length : 0)
      node.data.nb_parents += (task.is_part_of !== undefined ? task.is_part_of.length : 0)
      node.data.nb_children += (task.is_depending_on !== undefined ? task.is_depending_on.length : 0)
      node.data.nb_children += (task.has_part !== undefined ? task.has_part.length : 0)
      node.data.nb_related += (task.is_related_with !== undefined ? task.is_related_with.length : 0)

      // relationships
      if (task.is_depending_on !== undefined) {
        task.is_depending_on.forEach(other => {
          edges.push({
            data: {
              source: task.id,
              target: other,
              relation: "is_depending_on",
            },
          })
        })
      }
      if (task.is_blocking !== undefined) {
        task.is_blocking.forEach(other => {
          edges.push({
            data: {
              source: other,
              target: task.id,
              relation: "is_depending_on",
            },
          })
        })
      }
      if (task.is_related_with !== undefined) {
        task.is_related_with.forEach(other => {
          edges.push({
            data: {
              source: other,
              target: task.id,
              relation: "related_with",
            },
          })
        })
      }
      if (task.is_part_of !== undefined) {
        task.is_part_of.forEach(other => {
          edges.push({
            data: {
              source: task.id,
              target: other,
              relation: "part_of",
            },
          })
        })
      }
      if (task.has_part !== undefined) {
        task.has_part.forEach(other => {
          edges.push({
            data: {
              source: other,
              target: task.id,
              relation: "part_of",
            },
          })
        })
      }
      if (task.has_owner !== undefined) {
        node.data.parent = task.has_owner
      }
      if (task.has_milestone !== undefined) {
        node.data.parent = task.has_milestone
      }
      //if (task.has_author !== undefined) { edges.push({from: task.has_author, to: task.id}) }
      //if (task.has_assignee !== undefined) { task.has_assignee.forEach(other => edges.push({from: other, to: task.id})) }
      //if (task.has_reviewer !== undefined) { task.has_reviewer.forEach(other => edges.push({from: other, to: task.id})) }
      //if (task.has_label !== undefined) { task.has_label.forEach(other => edges.push({from: other, to: task.id})) }

      nodes.push(node)
    })

    nodes.forEach(node => {
      node.group = 'nodes'
      config.elements.push(node)
    })

    nodeHtmlLabel(cytoscape)
    cy = cytoscape(config)

    cy.on('tap', 'node', function(){
      try { // your browser may block popups
        window.open( this.data('id') );
      } catch(e){ // fall back on url change
        window.location.href = this.data('id');
      }
    });

    cy.nodeHtmlLabel(
      [
        {
          query: 'node.Issue, node.MergeRequest',
          halign: 'center',
          valign: 'center',
          halignBox: 'center',
          valignBox: 'center',
          cssClass: '',
          tpl: function(data){
            return Card(data);
          },
        },
      ],
    )

    var edgeMap = {}
    cy.batch(() => {
      edges.forEach(edge => {
        let isOk = true
        if (cy.getElementById(edge.data.source).empty()) {
          console.warn('missing node', edge.data.source)
          isOk = false
        }
        if (cy.getElementById(edge.data.target).empty()) {
          console.warn('missing node', edge.data.target)
          isOk = false
        }
        if (!isOk) {
          return
        }
        edge.group = 'edges'
        edge.data.id = edge.data.relation + edge.data.source + edge.data.target
        if (edge.data.id in edgeMap) {
          console.warn('duplicate edge', edge)
        } else {
          edgeMap[edge.data.id] = edge
          cy.add(edge)
        }
      })
    })

  }

  return (
    <div id="cy"></div>
  );
}

export default Visualizer;
