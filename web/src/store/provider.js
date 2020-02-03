import React, { useState } from "react";
import StoreContext from "./index";

const provider = props => {
  const [state, setState] = useState({
    apiData: null
  });
  return (
   <StoreContext.Provider
      value={{
        data: state,
        updateApiData: (data) => {
          setState({ ...state, apiData: data });
        }
      }}
    >
      {props.children}
    </StoreContext.Provider>
  );
};

export default provider;
