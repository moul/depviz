import React from "react";
import StoreContext from "../../store"

const HomePage = () => {
  return (
    <StoreContext.Consumer>
      {(context) => ( console.log("Ai context >>>", context))}
    </StoreContext.Consumer>
  )
}

export default HomePage;
