/* eslint-disable no-unused-vars */
/* eslint-disable no-useless-catch */

import React, { useEffect, useContext } from "react";
import { useForm } from "react-hook-form";
import { forEachObjIndexed } from "ramda";
import StoreContext from "../../store";
import { computeLayoutConfig } from "../../components/visualizer/utils"
import { fetchDepviz } from "../../api/depviz";
import "./menu.scss";

const Menu = () => {
  let AppStoreContext = useContext(StoreContext)
  const { updateApiData, updateLayout } = AppStoreContext
  const { register, getValues, setValue, handleSubmit } = useForm();
  let searchParams = new URLSearchParams(window.location.search)

  useEffect(() => {
    let urlData = {
      targets: searchParams.getAll("targets").join(",") || undefined,
      withClosed: searchParams.get("withClosed") || undefined,
      withIsolated: searchParams.get("withIsolated") || undefined,
      withPrs: searchParams.get("withPrs") || undefined,
      withExternalDeps: searchParams.get("withoutExternal-deps") || undefined,
      layout: searchParams.get("layout") || undefined
    }

    const setFormValue = (value, key) => {
      if (value) {
        setValue(key, value)
      }
    };

    forEachObjIndexed(setFormValue, urlData);

    if (urlData.targets) {
      try {
        makeAPICall(urlData);
      } catch(error) {
        throw error;
      }
    }
  }, [])

  const onSubmit = (data) => {
      makeAPICall(data);
  }

  const onRedraw = (layout) => {
    let layoutConfig = computeLayoutConfig(layout)
    let cyLayout = window.cy.layout(layoutConfig)
    cyLayout.run();
}

  const makeAPICall = async (data) => {
    const {
      targets,
      withClosed,
      withIsolated,
      withPrs,
      withExternalDeps,
      layout
    } = data;

    // construct url
    let url = `?${targets.split(",").map(target => `targets=${target.trim()}`).join("&")}&withClosed=${withClosed}&withIsolated=${withIsolated}&withPrs=${withPrs}&withoutExternal-deps=${withExternalDeps}&layout=${layout}`

    try {
      const response = await fetchDepviz(`/graph${url}`)
      updateApiData(response.data, layout)
      window.history.replaceState({} , "DepViz - Dependecy Visualization", url)
    } catch (error) {
      throw error;
    }
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <div className="form-group">
        <label htmlFor="targets">Repository:</label>
        <input ref={register} type="text" name="targets" id="targets" />
      </div>

      <div className="form-group">
        <input ref={register} type="checkbox" name="withClosed" id="withClosed" onChange={() => onSubmit(getValues())} />
        <label htmlFor="withClosed">Closed</label>

        <input ref={register} defaultChecked type="checkbox" name="withIsolated" id="withIsolated" onChange={() => onSubmit(getValues())} />
        <label htmlFor="withIsolated">Isolated</label>

        <input ref={register} defaultChecked type="checkbox" name="withPrs" id="withPrs" onChange={() => onSubmit(getValues())} />
        <label htmlFor="withPrs">PRs</label>

        <input ref={register} defaultChecked type="checkbox" name="withExternalDeps" id="withExternalDeps" onChange={() => onSubmit(getValues())} />
        <label htmlFor="withExternalDeps">Ext. Deps</label>
      </div>

      <div className="form-group">
        <label htmlFor="layout">Layout:</label>
        <select ref={register} name="layout" id="layout" onChange={e => updateLayout(e.target.value)}>
          <option value="circle">circle</option>
          <option value="cose">cose</option>
          <option value="breadthfirst">breadthfirst</option>
          <option value="concentric">concentric</option>
          <option value="grid">grid</option>
          <option value="random">random</option>
          <option value="cola">cola</option>
          <option value="elk">elk</option>
        </select>
      </div>

      <div className="button-group">
        <button type="submit">Generate</button>
        <button type="button" onClick={() => onRedraw(getValues().layout)}>Redraw</button>
      </div>
    </form>
  )
}

export default Menu;
