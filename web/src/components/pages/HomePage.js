import React from "react";
import StoreContext from "../../store";
import "./homepage.scss";

import Visualizer from "../visualizer";

const HomePage = () => {
  return (
    <StoreContext.Consumer>
      {({ data: { apiData, layout } }) => (
        <Visualizer data={apiData} layout={layout} />
      )}
    </StoreContext.Consumer>
  )
}

export default HomePage;
