import React, { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import 'tabler/js/tabler'
import html2canvas from 'html2canvas'
// import { bitmap2vector } from 'bitmap2vector'
import potrace from 'potrace'
import toBuffer from 'blob-to-buffer'
import { useStore } from '../../../hooks/useStore'
import { generateUrl, updateBrowserHistory } from './utils'
import fetchDepviz from '../../../api/depviz'

import './styles.scss'

const Menu = ({
  authToken, handleShowToken, urlParams = {},
}) => {
  const {
    updateApiData, updateLayout, updateLoadingGraph, layout, setShowInfoBox, updateUrlData,
  } = useStore()
  const {
    register, getValues, setValue, handleSubmit,
  } = useForm()

  const [urlData, setURLData] = useState(urlParams)
  const [showDropdown, setShowDropdown] = useState(false)
  const [waitingExport, setWaitingExport] = useState(false)

  // Initialize form data and make API call (only once)
  useEffect(() => {
    updateLayout(urlData.layout)
    if (urlData.targets) {
      updateLoadingGraph(true)
      // Process Timeline layout (disable all checkboxes except Closed)
      if (urlData.layout === 'timeline') {
        urlData.withClosed = true
        urlData.withoutIsolated = false
        urlData.withoutPrs = false
        urlData.withoutExternalDeps = false
        urlData.withFetch = false
        updateBrowserHistory(generateUrl(urlData))
        setURLData(urlData)
        setValue('withClosed', true)
        setValue('withoutIsolated', false)
        setValue('withoutPrs', false)
        setValue('withoutExternalDeps', false)
        setValue('withFetch', false)
      } else {
        Object.keys(urlData).map((key) => {
          if (urlData[key]) {
            setValue(key, urlData[key])
          }
        })
        urlData.withoutIsolated = !urlData.withoutIsolated
        urlData.withoutPrs = !urlData.withoutPrs
        urlData.withoutExternalDeps = !urlData.withoutExternalDeps
        urlData.withFetch = !urlData.withFetch
      }
      makeAPICall(urlData)
    }
  }, [])

  const makeAPICall = async (data) => {
    const response = await fetchDepviz(`/graph${generateUrl(data)}`)
    updateApiData(response.data, data.layout, data.targets)
    // updateBrowserHistory(url)
  }

  const handleURLData = (fetchApi = false, with_fetch = false) => {
    updateLoadingGraph(true)
    const data = getValues()
    const newUrlData = {
      ...urlData,
      ...data,
      withFetch: with_fetch,
    }
    newUrlData.withoutIsolated = !data.withoutIsolated
    newUrlData.withoutPrs = !data.withoutPrs
    newUrlData.withoutExternalDeps = !data.withoutExternalDeps
    updateBrowserHistory(generateUrl(newUrlData))
    setURLData(newUrlData)
    updateUrlData(newUrlData)
    if (fetchApi) {
      makeAPICall(newUrlData)
    }
  }

  const onSubmit = () => {
    handleURLData(true)
  }

  const handleFetch = () => {
    handleURLData(true, true)
  }

  const handleLayoutChange = () => {
    const data = getValues()
    handleURLData(true)
    // Process Timeline layout (disable all checkboxes except Closed)
    if (data.layout === 'timeline') {
      const newUrlData = {
        ...urlData,
        ...data,
        withFetch: false,
      }
      newUrlData.withClosed = true
      newUrlData.withoutIsolated = false
      newUrlData.withoutPrs = false
      newUrlData.withoutExternalDeps = false
      newUrlData.withFetch = false
      updateBrowserHistory(generateUrl(newUrlData))
      setURLData(newUrlData)
      // setValue('withClosed')
      setShowInfoBox(false)
      setValue('withClosed', true)
      setValue('withoutIsolated', false)
      setValue('withoutPrs', false)
      setValue('withoutExternalDeps', false)
      setValue('withFetch', false)
    }
    updateLayout(data.layout)
  }

  const handleCheckboxChange = () => {
    setShowInfoBox(false)
    handleURLData(true)
    // handleRedraw()
  }

  const handleRedraw = () => {
    if (window.cy) {
      window.cy.layout(layout).run()
    }
  }

  const saveGraph = (exportType) => async (e) => {
    e.preventDefault()
    // Prevent multiple clicks
    if (waitingExport) {
      return
    }
    setShowDropdown(false)
    setWaitingExport(true)

    const selector = document.getElementById('cy')
    const appendTo = document.getElementById('canvas-test')
    const canvasElem = document.getElementById('exported-canvas') || null

    let type = ''
    // const scaleRatio = 10
    // const ctxOrig = canvas.getContext('2d')
    // let canvasW = 500
    // let canvasH = 300
    switch (exportType) {
      case 'svg':
        type = 'image/svg'
        /* ctxOrig.scale(scaleRatio, scaleRatio)
        canvasW = canvas.width
        canvasH = canvas.height
        if (!canvasW && !canvasH) {
          canvasW = canvas.getBoundingClientRect().width
          canvasH = canvas.getBoundingClientRect().height
        }
        canvasW *= scaleRatio
        canvasH *= scaleRatio
        canvas.width = canvasW
        canvas.height = canvasH */
        break
      case 'jpg':
        type = 'image/jpeg'
        break
      default:
        type = 'image/png'
        break
    }

    const canvas = await html2canvas(selector, {
      backgroundColor: exportType === 'jpg' ? '#FFFFFF' : null,
      windowWidth: selector.scrollWidth,
      windowHeight: selector.scrollHeight,
      // windowWidth: exportType === 'svg' ? selector.scrollWidth * scaleRatio : selector.scrollWidth,
      // windowHeight: exportType === 'svg' ? selector.scrollHeight * scaleRatio : selector.scrollHeight,
    })

    if (!appendTo) {
      document.body.appendChild(canvas)
    } else {
      if (canvasElem) {
        appendTo.removeChild(canvasElem)
      }
      appendTo.appendChild(canvas)
    }

    if (exportType === 'svg') { // Export to SVG
      canvas.toBlob((blob) => {
        const newImg = document.createElement('img')
        const url = URL.createObjectURL(blob)

        newImg.onload = () => {
          URL.revokeObjectURL(url)
        }

        newImg.src = url

        // Convert Blob to Buffer
        toBuffer(blob, async (err, buffer) => {
          if (err) throw err
          // WARNING: slow down rendering a lot, but more quality, for options details https://github.com/jankovicsandras/imagetracerjs/blob/master/options.md
          /* const { content } = await bitmap2vector({
            input: buffer,
            layering: 1,
            // detailed preset
            pathomit: 0,
            roundcoords: 2,
            ltres: 0.5,
            qtres: 0.5,
            numberofcolors: 64,
          })
          downloadSVG(content, exportType) */
          /* const result = await png2svg({
            tracer: 'imagetracer',
            optimize: true,
            input: buffer,
            numberofcolors: 24,
            pathomit: 1,
          }) */
          const params = {
            turdSize: 0.5,
            alphaMax: 0.1,
            optCurve: true,
            optTolerance: 0.2,
            threshold: 240,
            blackOnWhite: true,
            color: '#20D6B5',
          }
          potrace.trace(buffer, params, (err, svg) => {
            console.log('svg: ', svg)
            downloadSVG(svg, exportType)
          })
        })
      }, 'image/png')
    } else {
      canvas.toBlob((blob) => {
        const newImg = document.createElement('img')
        const url = URL.createObjectURL(blob)

        newImg.onload = () => {
          URL.revokeObjectURL(url)
        }

        newImg.src = url
        const a = document.getElementById('downloadgraph')
        a.href = url
        const currDate = new Date()
        const currDay = currDate.getDate()
        const currMonth = currDate.getMonth()
        const currYear = currDate.getFullYear()
        a.download = `depviz-${layout.name}-graph-${currMonth + 1}-${currDay}-${currYear}.${exportType}`
        a.click()
        setWaitingExport(false)
      }, type, 1)
    }
  }

  const downloadSVG = (svg, exportType) => {
    let source = svg
    if (!source.match(/^<svg[^>]+xmlns="http:\/\/www\.w3\.org\/2000\/svg"/)) {
      source = source.replace(/^<svg/, '<svg xmlns="http://www.w3.org/2000/svg"')
    }
    if (!source.match(/^<svg[^>]+"http:\/\/www\.w3\.org\/1999\/xlink"/)) {
      source = source.replace(/^<svg/, '<svg xmlns:xlink="http://www.w3.org/1999/xlink"')
    }

    // add xml declaration
    source = `<?xml version="1.0" standalone="no"?>\r\n${source}`

    // convert svg source to URI data scheme. */
    const url = `data:image/svg+xml;charset=utf-8,${encodeURIComponent(source)}`
    const a = document.getElementById('downloadgraph')
    a.href = url
    const currDate = new Date()
    const currDay = currDate.getDate()
    const currMonth = currDate.getMonth()
    const currYear = currDate.getFullYear()
    a.download = `depviz-${layout.name}-graph-${currMonth + 1}-${currDay}-${currYear}.${exportType}`
    a.click()
    setWaitingExport(false)
  }

  return (
    <div className="header d-lg-flex p-3">
      <div className="container">
        <form onSubmit={handleSubmit(onSubmit)} className="row align-items-center">
          <div className="col-lg-6 order-lg-first">
            <div className="form-group repo-and-token">
              <label htmlFor="targets" className="form-label">
                <div className="input-group">
                  <input ref={register} type="text" name="targets" id="targets" placeholder="Repository" className="form-control" />
                  <div className="input-group-append">
                    <button type="submit" className="btn btn-primary ml-auto">Generate</button>
                    <button type="button" onClick={handleRedraw} className="btn btn-secondary ml-auto">Redraw</button>
                    <button type="button" onClick={handleFetch} className="btn btn-secondary ml-auto">Fetch</button>
                  </div>
                </div>
              </label>
              <a id="downloadgraph" style={{ display: 'none' }} />

              <div className="dropdown">
                <a className={waitingExport ? 'btn btn-info dropdown-toggle disabled' : 'btn btn-info dropdown-toggle'} href="#" role="button" id="dropdownMenuLink" data-toggle="dropdown" aria-haspopup="true" aria-expanded="false" onClick={() => setShowDropdown(!showDropdown)}>
                  {waitingExport ? 'Exporting...' : 'Export'}
                </a>
                <div className={showDropdown ? 'dropdown-menu show' : 'dropdown-menu'} aria-labelledby="dropdownMenuLink">
                  <a className="dropdown-item" href="#" onClick={saveGraph('png')}>Save as PNG</a>
                  <a className="dropdown-item" href="#" onClick={saveGraph('jpg')}>Save as JPG</a>
                  <a className="dropdown-item" href="#" onClick={saveGraph('svg')}>Save as SVG (beta)</a>
                </div>
              </div>

              <button onClick={handleShowToken} className="btn">
                {authToken ? 'Change token' : '+ Add token'}
              </button>
            </div>

          </div>
          <div className="col-lg ml-right">
            <div className="form-group">
              <label htmlFor="withClosed" className="custom-control custom-checkbox custom-control-inline">
                <input ref={register} type="checkbox" name="withClosed" id="withClosed" onChange={handleCheckboxChange} disabled={layout.name === 'timeline'} className="custom-control-input" />
                <span className="custom-control-label">Closed</span>
              </label>

              <label htmlFor="withoutIsolated" className="custom-control custom-checkbox custom-control-inline">
                <input ref={register} type="checkbox" name="withoutIsolated" id="withoutIsolated" onChange={handleCheckboxChange} disabled={layout.name === 'timeline'} className="custom-control-input" />
                <span className="custom-control-label">Isolated</span>
              </label>

              <label htmlFor="withoutPrs" className="custom-control custom-checkbox custom-control-inline">
                <input ref={register} type="checkbox" name="withoutPrs" id="withoutPrs" onChange={handleCheckboxChange} disabled={layout.name === 'timeline'} className="custom-control-input" />
                <span className="custom-control-label">PRs</span>
              </label>

              <label htmlFor="withoutExternalDeps" className="custom-control custom-checkbox custom-control-inline">
                <input ref={register} type="checkbox" name="withoutExternalDeps" id="withoutExternalDeps" onChange={handleCheckboxChange} disabled={layout.name === 'timeline'} className="custom-control-input" />
                <span className="custom-control-label">Ext. Deps</span>
              </label>
            </div>

            <div className="form-group layout-select">
              <label htmlFor="layout">
                <span className="custom-control">Layout:</span>
                <select ref={register} name="layout" id="layout" onChange={handleLayoutChange} className="form-control custom-select selectized">
                  <option value="circle">circle</option>
                  <option value="cose">cose</option>
                  <option value="breadthfirst">breadthfirst</option>
                  <option value="concentric">concentric</option>
                  <option value="grid">grid</option>
                  <option value="random">random</option>
                  <option value="cola">cola</option>
                  <option value="elk">elk</option>
                  <option value="gantt">gantt</option>
                  <option value="flow">flow</option>
                  <option value="timeline">timeline</option>
                </select>
              </label>
            </div>
          </div>
        </form>
      </div>
      <canvas id="imgcanvas" style={{ display: 'none' }} />
    </div>
  )
}

export default Menu
