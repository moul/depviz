import React from "react";
import { useForm } from "react-hook-form";
import "./menu.scss"

const Menu = () => {
  const { register, handleSubmit } = useForm();

  let cy;
  let layoutConfig
  let template

  const onSubmit = data => {

      const {
        targets,
        withClosed,
        withIsolated,
        withPrs,
        withExternalDeps
      } = data;

      // construct url
      let url = `https://depviz-demo.moul.io/api/graph?targets=${targets}&withClosed=${withClosed}&withIsolated=${withIsolated}&withPrs=${withPrs}&withoutExternal-deps=${withExternalDeps}`

    console.log(data)
    console.log(url)
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <div className="form-group">
        <label htmlFor="targets">Repository:</label>
        <input ref={register} type="text" name="targets" id="targets" />
      </div>

      <div className="form-group">
        <input ref={register} type="checkbox" name="withClosed" id="withClosed" />
        <label htmlFor="withClosed">Closed</label>

        <input ref={register} defaultChecked type="checkbox" name="withIsolated" id="withIsolated" />
        <label htmlFor="withIsolated">Isolated</label>

        <input ref={register} defaultChecked type="checkbox" name="withPrs" id="withPrs" />
        <label htmlFor="withPrs">PRs</label>

        <input ref={register} defaultChecked type="checkbox" name="withExternalDeps" id="withExternalDeps" />
        <label htmlFor="withExternalDeps">Ext. Deps</label>
      </div>

      <div className="form-group">
        <label htmlFor="layout">Layout:</label>
        <select ref={register} name="layout" id="layout">
          <option value="circle">circle</option>
          <option value="cose">cose</option>
          <option value="breadthfirst">breadthfirst</option>
          <option value="concetric">concetric</option>
          <option value="grid">grid</option>
          <option value="random">random</option>
          <option value="cola">cola</option>
          <option value="elk">elk</option>

        </select>
      </div>

      <div className="button-group">
        <button type="button">Redraw</button>
        <button type="submit">Generate</button>
      </div>
    </form>
  )
}

export default Menu;
