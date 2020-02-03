/* eslint-disable react/prop-types */

import React, { useState } from "react";
import StoreContext from "./index";

const provider = props => {
  const [state, setState] = useState({
    apiData: null,
    layout: null
  });
  return (
   <StoreContext.Provider
      value={{
        data: state,
        updateApiData: (data, layout) => {
          setState({ ...state, apiData: data, layout: layout});
        }
      }}
    >
      {props.children}
    </StoreContext.Provider>
  );
};

export default provider;
