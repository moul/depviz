import React, {
  createContext, useContext, useMemo, useState,
} from 'react'

const testData = [
  {
    id: 'https://github.com/moul-bot/depviz-test/issues/5',
    created_at: '2019-08-08T18:55:47Z',
    updated_at: '2019-08-08T18:55:47Z',
    local_id: 'moul-bot/depviz-test#5',
    kind: 'Issue',
    title: 'Issue 5',
    description: 'Depends on #4',
    driver: 'GitHub',
    state: 'Open',
    has_author: 'https://github.com/moul-bot',
    has_owner: 'https://github.com/moul-bot/depviz-test',
    is_depending_on: [
      'https://github.com/moul-bot/depviz-test/issues/4',
    ],
  },
  {
    id: 'https://github.com/moul-bot/depviz-test/issues/7',
    created_at: '2019-08-08T18:56:14Z',
    updated_at: '2019-09-03T09:07:03Z',
    local_id: 'moul-bot/depviz-test#7',
    kind: 'Issue',
    title: 'Issue 7',
    description: 'Depends on #4\r\nDepends on https://github.com/moul/depviz-test/milestone/1',
    driver: 'GitHub',
    state: 'Open',
    has_author: 'https://github.com/moul-bot',
    has_owner: 'https://github.com/moul-bot/depviz-test',
    is_depending_on: [
      'https://github.com/moul-bot/depviz-test/issues/4',
      'https://github.com/moul/depviz-test/milestone/1',
    ],
  },
  {
    id: 'https://github.com/moul/depviz-test/issues/1',
    created_at: '2019-08-06T15:35:49Z',
    updated_at: '2019-08-06T15:35:49Z',
    local_id: 'moul/depviz-test#1',
    kind: 'Issue',
    title: "I'm a standard issue",
    driver: 'GitHub',
    state: 'Open',
    has_author: 'https://github.com/moul',
    has_owner: 'https://github.com/moul/depviz-test',
  },
  {
    id: 'https://github.com/moul/depviz-test/issues/10',
    created_at: '2019-09-03T08:51:47Z',
    updated_at: '2019-12-03T17:35:06Z',
    local_id: 'moul/depviz-test#10',
    kind: 'Issue',
    title: 'New test',
    description: 'Depends on #4 \r\nDepends on #6 \r\nBlocks #7 \r\nDepends on https://github.com/moul-bot/depviz-test/issues/5',
    driver: 'GitHub',
    state: 'Open',
    has_author: 'https://github.com/moul',
    has_owner: 'https://github.com/moul/depviz-test',
    is_depending_on: [
      'https://github.com/moul-bot/depviz-test/issues/5',
      'https://github.com/moul/depviz-test/issues/4',
      'https://github.com/moul/depviz-test/issues/6',
    ],
    is_blocking: [
      'https://github.com/moul/depviz-test/issues/7',
    ],
  },
  {
    id: 'https://github.com/moul/depviz-test/issues/10',
    created_at: '2019-09-03T08:51:47Z',
    updated_at: '2019-12-03T17:35:06Z',
    local_id: 'moul/depviz-test#10',
    kind: 'Issue',
    title: 'New test',
    description: 'Depends on #4 \r\nDepends on #6 \r\nBlocks #7 \r\nDepends on https://github.com/moul-bot/depviz-test/issues/5',
    driver: 'GitHub',
    state: 'Open',
    has_author: 'https://github.com/moul',
    has_owner: 'https://github.com/moul/depviz-test',
    is_depending_on: [
      'https://github.com/moul-bot/depviz-test/issues/5',
      'https://github.com/moul/depviz-test/issues/4',
      'https://github.com/moul/depviz-test/issues/6',
    ],
    is_blocking: [
      'https://github.com/moul/depviz-test/issues/7',
    ],
  },
  {
    id: 'https://github.com/moul/depviz-test/issues/10',
    created_at: '2019-09-03T08:51:47Z',
    updated_at: '2019-12-03T17:35:06Z',
    local_id: 'moul/depviz-test#10',
    kind: 'Issue',
    title: 'New test',
    description: 'Depends on #4 \r\nDepends on #6 \r\nBlocks #7 \r\nDepends on https://github.com/moul-bot/depviz-test/issues/5',
    driver: 'GitHub',
    state: 'Open',
    has_author: 'https://github.com/moul',
    has_owner: 'https://github.com/moul/depviz-test',
    is_depending_on: [
      'https://github.com/moul-bot/depviz-test/issues/5',
      'https://github.com/moul/depviz-test/issues/4',
      'https://github.com/moul/depviz-test/issues/6',
    ],
    is_blocking: [
      'https://github.com/moul/depviz-test/issues/7',
    ],
  },
  {
    id: 'https://github.com/moul/depviz-test/issues/10',
    created_at: '2019-09-03T08:51:47Z',
    updated_at: '2019-12-03T17:35:06Z',
    local_id: 'moul/depviz-test#10',
    kind: 'Issue',
    title: 'New test',
    description: 'Depends on #4 \r\nDepends on #6 \r\nBlocks #7 \r\nDepends on https://github.com/moul-bot/depviz-test/issues/5',
    driver: 'GitHub',
    state: 'Open',
    has_author: 'https://github.com/moul',
    has_owner: 'https://github.com/moul/depviz-test',
    is_depending_on: [
      'https://github.com/moul-bot/depviz-test/issues/5',
      'https://github.com/moul/depviz-test/issues/4',
      'https://github.com/moul/depviz-test/issues/6',
    ],
    is_blocking: [
      'https://github.com/moul/depviz-test/issues/7',
    ],
  },
  {
    id: 'https://github.com/moul/depviz-test/issues/11',
    created_at: '2019-11-04T11:59:52Z',
    updated_at: '2019-11-04T12:05:12Z',
    local_id: 'moul/depviz-test#11',
    kind: 'Issue',
    title: 'test short names',
    description: 'https://github.com/moul/depviz-test/issues/2\r\nhttps://github.com/moul/depviz-test/issues/10\r\nhttps://github.com/moul/depviz-test-two-issues/pull/4\r\nhttps://github.com/moul/depviz-test-two-issues/pull/3\r\nhttps://github.com/moul/depviz-test-two-issues/projects/1#card-28544696\r\nhttps://github.com/moul/depviz-test-two-issues/milestone/1\r\nhttps://github.com/moul\r\nhttps://github.com/berty',
    driver: 'GitHub',
    state: 'Open',
    has_author: 'https://github.com/moul',
    has_owner: 'https://github.com/moul/depviz-test',
  },
  {
    id: 'https://github.com/moul/depviz-test/issues/2',
    created_at: '2019-08-06T15:36:09Z',
    updated_at: '2019-10-29T08:59:41Z',
    local_id: 'moul/depviz-test#2',
    kind: 'Issue',
    title: "I'm an issue with a milestone, some projects, and some labels",
    driver: 'GitHub',
    completed_at: '2019-10-29T08:59:41Z',
    state: 'Closed',
    has_author: 'https://github.com/moul',
    has_owner: 'https://github.com/moul/depviz-test',
    has_milestone: 'https://github.com/moul/depviz-test/milestone/1',
    has_label: [
      'https://github.com/moul/depviz-test/labels/bug',
      'https://github.com/moul/depviz-test/labels/documentation',
      'https://github.com/moul/depviz-test/labels/enhancement',
    ],
  },
  {
    id: 'https://github.com/moul/depviz-test/issues/2',
    created_at: '2019-08-06T15:36:09Z',
    updated_at: '2019-10-29T08:59:41Z',
    local_id: 'moul/depviz-test#2',
    kind: 'Issue',
    title: "I'm an issue with a milestone, some projects, and some labels",
    driver: 'GitHub',
    completed_at: '2019-10-29T08:59:41Z',
    state: 'Closed',
    has_author: 'https://github.com/moul',
    has_owner: 'https://github.com/moul/depviz-test',
    has_milestone: 'https://github.com/moul/depviz-test/milestone/1',
    has_label: [
      'https://github.com/moul/depviz-test/labels/bug',
      'https://github.com/moul/depviz-test/labels/documentation',
      'https://github.com/moul/depviz-test/labels/enhancement',
    ],
  },
  {
    id: 'https://github.com/moul/depviz-test/issues/2',
    created_at: '2019-08-06T15:36:09Z',
    updated_at: '2019-10-29T08:59:41Z',
    local_id: 'moul/depviz-test#2',
    kind: 'Issue',
    title: "I'm an issue with a milestone, some projects, and some labels",
    driver: 'GitHub',
    completed_at: '2019-10-29T08:59:41Z',
    state: 'Closed',
    has_author: 'https://github.com/moul',
    has_owner: 'https://github.com/moul/depviz-test',
    has_milestone: 'https://github.com/moul/depviz-test/milestone/1',
    has_label: [
      'https://github.com/moul/depviz-test/labels/bug',
      'https://github.com/moul/depviz-test/labels/documentation',
      'https://github.com/moul/depviz-test/labels/enhancement',
    ],
  },
  {
    id: 'https://github.com/moul/depviz-test/issues/3',
    created_at: '2019-08-06T15:36:45Z',
    updated_at: '2019-08-06T15:36:45Z',
    local_id: 'moul/depviz-test#3',
    kind: 'Issue',
    title: "I'm an issue that depends on another",
    description: 'Depends on #2 ',
    driver: 'GitHub',
    state: 'Open',
    has_author: 'https://github.com/moul',
    has_owner: 'https://github.com/moul/depviz-test',
    is_depending_on: [
      'https://github.com/moul/depviz-test/issues/2',
    ],
  },
  {
    id: 'https://github.com/moul/depviz-test/issues/3',
    created_at: '2019-08-06T15:36:45Z',
    updated_at: '2019-08-06T15:36:45Z',
    local_id: 'moul/depviz-test#3',
    kind: 'Issue',
    title: "I'm an issue that depends on another",
    description: 'Depends on #2 ',
    driver: 'GitHub',
    state: 'Open',
    has_author: 'https://github.com/moul',
    has_owner: 'https://github.com/moul/depviz-test',
    is_depending_on: [
      'https://github.com/moul/depviz-test/issues/2',
    ],
  },
  {
    id: 'https://github.com/moul/depviz-test/issues/4',
    created_at: '2019-08-06T15:36:58Z',
    updated_at: '2019-08-06T15:36:58Z',
    local_id: 'moul/depviz-test#4',
    kind: 'Issue',
    title: "I'm an issue that also depends on another",
    description: 'Depends on #2',
    driver: 'GitHub',
    state: 'Open',
    has_author: 'https://github.com/moul',
    has_owner: 'https://github.com/moul/depviz-test',
    is_depending_on: [
      'https://github.com/moul/depviz-test/issues/2',
    ],
  },
  {
    id: 'https://github.com/moul/depviz-test/issues/4',
    created_at: '2019-08-06T15:36:58Z',
    updated_at: '2019-08-06T15:36:58Z',
    local_id: 'moul/depviz-test#4',
    kind: 'Issue',
    title: "I'm an issue that also depends on another",
    description: 'Depends on #2',
    driver: 'GitHub',
    state: 'Open',
    has_author: 'https://github.com/moul',
    has_owner: 'https://github.com/moul/depviz-test',
    is_depending_on: [
      'https://github.com/moul/depviz-test/issues/2',
    ],
  },
  {
    id: 'https://github.com/moul/depviz-test/issues/5',
    created_at: '2019-08-06T15:37:21Z',
    updated_at: '2019-10-29T08:59:38Z',
    local_id: 'moul/depviz-test#5',
    kind: 'Issue',
    title: "I'm an issue that depends on multiple issues",
    description: 'Depends on #4 \r\nDepends on #3 ',
    driver: 'GitHub',
    completed_at: '2019-10-29T08:59:38Z',
    state: 'Closed',
    has_author: 'https://github.com/moul',
    has_owner: 'https://github.com/moul/depviz-test',
    is_depending_on: [
      'https://github.com/moul/depviz-test/issues/3',
      'https://github.com/moul/depviz-test/issues/4',
    ],
  },
  {
    id: 'https://github.com/moul/depviz-test/issues/5',
    created_at: '2019-08-06T15:37:21Z',
    updated_at: '2019-10-29T08:59:38Z',
    local_id: 'moul/depviz-test#5',
    kind: 'Issue',
    title: "I'm an issue that depends on multiple issues",
    description: 'Depends on #4 \r\nDepends on #3 ',
    driver: 'GitHub',
    completed_at: '2019-10-29T08:59:38Z',
    state: 'Closed',
    has_author: 'https://github.com/moul',
    has_owner: 'https://github.com/moul/depviz-test',
    is_depending_on: [
      'https://github.com/moul/depviz-test/issues/3',
      'https://github.com/moul/depviz-test/issues/4',
    ],
  },
  {
    id: 'https://github.com/moul/depviz-test/issues/5',
    created_at: '2019-08-06T15:37:21Z',
    updated_at: '2019-10-29T08:59:38Z',
    local_id: 'moul/depviz-test#5',
    kind: 'Issue',
    title: "I'm an issue that depends on multiple issues",
    description: 'Depends on #4 \r\nDepends on #3 ',
    driver: 'GitHub',
    completed_at: '2019-10-29T08:59:38Z',
    state: 'Closed',
    has_author: 'https://github.com/moul',
    has_owner: 'https://github.com/moul/depviz-test',
    is_depending_on: [
      'https://github.com/moul/depviz-test/issues/3',
      'https://github.com/moul/depviz-test/issues/4',
    ],
  },
  {
    id: 'https://github.com/moul/depviz-test/issues/6',
    created_at: '2019-08-06T15:37:44Z',
    updated_at: '2019-08-06T15:37:44Z',
    local_id: 'moul/depviz-test#6',
    kind: 'Issue',
    title: "I'm an issue that depends on the same issue at different levels",
    description: 'Depends on #2 \r\nDepends on #3 \r\nDepends on #5 ',
    driver: 'GitHub',
    state: 'Open',
    has_author: 'https://github.com/moul',
    has_owner: 'https://github.com/moul/depviz-test',
    is_depending_on: [
      'https://github.com/moul/depviz-test/issues/2',
      'https://github.com/moul/depviz-test/issues/3',
      'https://github.com/moul/depviz-test/issues/5',
    ],
  },
  {
    id: 'https://github.com/moul/depviz-test/issues/6',
    created_at: '2019-08-06T15:37:44Z',
    updated_at: '2019-08-06T15:37:44Z',
    local_id: 'moul/depviz-test#6',
    kind: 'Issue',
    title: "I'm an issue that depends on the same issue at different levels",
    description: 'Depends on #2 \r\nDepends on #3 \r\nDepends on #5 ',
    driver: 'GitHub',
    state: 'Open',
    has_author: 'https://github.com/moul',
    has_owner: 'https://github.com/moul/depviz-test',
    is_depending_on: [
      'https://github.com/moul/depviz-test/issues/2',
      'https://github.com/moul/depviz-test/issues/3',
      'https://github.com/moul/depviz-test/issues/5',
    ],
  },
  {
    id: 'https://github.com/moul/depviz-test/issues/6',
    created_at: '2019-08-06T15:37:44Z',
    updated_at: '2019-08-06T15:37:44Z',
    local_id: 'moul/depviz-test#6',
    kind: 'Issue',
    title: "I'm an issue that depends on the same issue at different levels",
    description: 'Depends on #2 \r\nDepends on #3 \r\nDepends on #5 ',
    driver: 'GitHub',
    state: 'Open',
    has_author: 'https://github.com/moul',
    has_owner: 'https://github.com/moul/depviz-test',
    is_depending_on: [
      'https://github.com/moul/depviz-test/issues/2',
      'https://github.com/moul/depviz-test/issues/3',
      'https://github.com/moul/depviz-test/issues/5',
    ],
  },
  {
    id: 'https://github.com/moul/depviz-test/issues/6',
    created_at: '2019-08-06T15:37:44Z',
    updated_at: '2019-08-06T15:37:44Z',
    local_id: 'moul/depviz-test#6',
    kind: 'Issue',
    title: "I'm an issue that depends on the same issue at different levels",
    description: 'Depends on #2 \r\nDepends on #3 \r\nDepends on #5 ',
    driver: 'GitHub',
    state: 'Open',
    has_author: 'https://github.com/moul',
    has_owner: 'https://github.com/moul/depviz-test',
    is_depending_on: [
      'https://github.com/moul/depviz-test/issues/2',
      'https://github.com/moul/depviz-test/issues/3',
      'https://github.com/moul/depviz-test/issues/5',
    ],
  },
  {
    id: 'https://github.com/moul/depviz-test/issues/7',
    created_at: '2019-08-06T15:38:05Z',
    updated_at: '2019-11-19T17:30:28Z',
    local_id: 'moul/depviz-test#7',
    kind: 'Issue',
    title: "I'm an issue that depends on an issue that itself depends on multiple ones",
    description: 'Depends on #6 ',
    driver: 'GitHub',
    state: 'Open',
    has_author: 'https://github.com/moul',
    has_owner: 'https://github.com/moul/depviz-test',
    has_milestone: 'https://github.com/moul/depviz-test/milestone/1',
    is_depending_on: [
      'https://github.com/moul/depviz-test/issues/6',
    ],
  },
  {
    id: 'https://github.com/moul/depviz-test/issues/7',
    created_at: '2019-08-06T15:38:05Z',
    updated_at: '2019-11-19T17:30:28Z',
    local_id: 'moul/depviz-test#7',
    kind: 'Issue',
    title: "I'm an issue that depends on an issue that itself depends on multiple ones",
    description: 'Depends on #6 ',
    driver: 'GitHub',
    state: 'Open',
    has_author: 'https://github.com/moul',
    has_owner: 'https://github.com/moul/depviz-test',
    has_milestone: 'https://github.com/moul/depviz-test/milestone/1',
    is_depending_on: [
      'https://github.com/moul/depviz-test/issues/6',
    ],
  },
  {
    id: 'https://github.com/moul/depviz-test/issues/7',
    created_at: '2019-08-06T15:38:05Z',
    updated_at: '2019-11-19T17:30:28Z',
    local_id: 'moul/depviz-test#7',
    kind: 'Issue',
    title: "I'm an issue that depends on an issue that itself depends on multiple ones",
    description: 'Depends on #6 ',
    driver: 'GitHub',
    state: 'Open',
    has_author: 'https://github.com/moul',
    has_owner: 'https://github.com/moul/depviz-test',
    has_milestone: 'https://github.com/moul/depviz-test/milestone/1',
    is_depending_on: [
      'https://github.com/moul/depviz-test/issues/6',
    ],
  },
  {
    id: 'https://github.com/moul/depviz-test/issues/8',
    created_at: '2019-08-06T15:40:58Z',
    updated_at: '2019-11-19T17:30:28Z',
    local_id: 'moul/depviz-test#8',
    kind: 'Issue',
    title: 'An issue in an isolated group of 2',
    driver: 'GitHub',
    state: 'Open',
    has_author: 'https://github.com/moul',
    has_owner: 'https://github.com/moul/depviz-test',
    has_milestone: 'https://github.com/moul/depviz-test/milestone/1',
  },
  {
    id: 'https://github.com/moul/depviz-test/issues/8',
    created_at: '2019-08-06T15:40:58Z',
    updated_at: '2019-11-19T17:30:28Z',
    local_id: 'moul/depviz-test#8',
    kind: 'Issue',
    title: 'An issue in an isolated group of 2',
    driver: 'GitHub',
    state: 'Open',
    has_author: 'https://github.com/moul',
    has_owner: 'https://github.com/moul/depviz-test',
    has_milestone: 'https://github.com/moul/depviz-test/milestone/1',
  },
  {
    id: 'https://github.com/moul/depviz-test/issues/9',
    created_at: '2019-08-06T15:41:14Z',
    updated_at: '2019-11-19T17:30:28Z',
    local_id: 'moul/depviz-test#9',
    kind: 'Issue',
    title: 'Another issue in an isolated group of 2',
    description: 'Depends on #8',
    driver: 'GitHub',
    state: 'Open',
    has_author: 'https://github.com/moul',
    has_owner: 'https://github.com/moul/depviz-test',
    has_milestone: 'https://github.com/moul/depviz-test/milestone/1',
    is_depending_on: [
      'https://github.com/moul/depviz-test/issues/8',
    ],
  },
  {
    id: 'https://github.com/moul/depviz-test/issues/9',
    created_at: '2019-08-06T15:41:14Z',
    updated_at: '2019-11-19T17:30:28Z',
    local_id: 'moul/depviz-test#9',
    kind: 'Issue',
    title: 'Another issue in an isolated group of 2',
    description: 'Depends on #8',
    driver: 'GitHub',
    state: 'Open',
    has_author: 'https://github.com/moul',
    has_owner: 'https://github.com/moul/depviz-test',
    has_milestone: 'https://github.com/moul/depviz-test/milestone/1',
    is_depending_on: [
      'https://github.com/moul/depviz-test/issues/8',
    ],
  },
  {
    id: 'https://github.com/moul/depviz-test/milestone/1',
    created_at: '2019-08-06T15:36:28Z',
    updated_at: '2019-11-19T17:30:28Z',
    local_id: 'moul/depviz-test/milestone/1',
    kind: 'Milestone',
    title: 'lorem-ipsum-milestone',
    driver: 'GitHub',
    state: 'Open',
    has_author: 'https://github.com/moul',
    has_owner: 'https://github.com/moul/depviz-test',
  },
]

const DEFAULT_STATE = {
  apiData: {
    tasks: testData,
  },
  layout: {
    name: 'gantt',
    avoidOverlap: true,
  },
  repName: 'moul-bot/depviz-test',
}

function createContextValue(state, setState) {
  let layoutConfig = {}
  const computeLayoutConfig = (layout) => {
    switch (layout) {
      case 'circle':
        layoutConfig = {
          name: 'circle',
          avoidOverlap: true,
        }
        break
      case 'cose':
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
        break
      case 'breadthfirst':
        layoutConfig = {
          name: 'breadthfirst',
        }
        break
      case 'concentric':
        layoutConfig = {
          name: 'concentric',
        }
        break
      case 'grid':
        layoutConfig = {
          name: 'grid',
          condense: true,
        }
        break
      case 'random':
        layoutConfig = {
          name: 'random',
        }
        break
      case 'cola':
        layoutConfig = {
          name: 'cola',
          animate: false,
          refresh: 1,
          padding: 30,
          maxSimulationTime: 100,
        }
        break
      case 'elk':
        layoutConfig = {
          name: 'elk',
          elk: {
            zoomToFit: true,
            algorithm: 'mrtree',
            separateConnectedComponents: false,
          },
        }
        break
      case 'gantt':
        layoutConfig = {
          name: 'gantt',
        }
        break
      case 'flow':
        layoutConfig = {
          name: 'flow',
        }
        break
      default:
        break
    }

    return layoutConfig
  }
  return {
    ...state,
    updateApiData: (data, layout, repName) => {
      setState({
        ...state, apiData: data, layout: computeLayoutConfig(layout), repName,
      })
    },
    updateLayout: (layout) => {
      setState({ ...state, layout: computeLayoutConfig(layout) })
    },
  }
}

const StoreContext = createContext(createContextValue({
  ...DEFAULT_STATE,
  setState: () => console.error('You are using StoreContext without StoreProvider!'),
}))

export function useStore() {
  return useContext(StoreContext)
}

export function StoreProvider({ context, children }) {
  // console.log('authContext: ', context)
  const [state, setState] = useState({
    ...DEFAULT_STATE,
    ...context,
  })

  // Memoize context values
  const contextValue = useMemo(() => createContextValue(state, setState), [state, setState])

  return (<StoreContext.Provider value={contextValue}>{children}</StoreContext.Provider>)
}
