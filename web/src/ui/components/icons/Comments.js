import React from 'react'

const CommentsIcon = () => (
  <svg width="28px" height="25px" viewBox="0 0 28 25" version="1.1" xmlns="http://www.w3.org/2000/svg">
    <defs>
      <rect x="0" y="0" width="478" height="255" rx="15" />
      <filter x="-4.8%" y="-7.5%" width="109.6%" height="118.0%" filterUnits="objectBoundingBox">
        <feMorphology radius="1" operator="dilate" in="SourceAlpha" result="shadowSpreadOuter1" />
        <feOffset dx="0" dy="4" in="shadowSpreadOuter1" result="shadowOffsetOuter1" />
        <feGaussianBlur stdDeviation="6" in="shadowOffsetOuter1" result="shadowBlurOuter1" />
        <feColorMatrix values="0 0 0 0 0.219607843   0 0 0 0 0.231372549   0 0 0 0 0.384313725  0 0 0 0.15 0" type="matrix" in="shadowBlurOuter1" />
      </filter>
    </defs>
    <g stroke="none" strokeWidth="1" fill="none" fillRule="evenodd">
      <g transform="translate(-346.000000, -22.000000)">
        <g transform="translate(62.000000, 21.000000)" fill="#DFE0EB" fillRule="nonzero">
          <g className="icon-comments" transform="translate(284.000000, 1.000000)">
            <path d="M20.2222222,10.9363714 L20.2222222,3.12467755 C20.2222222,1.40122259 18.8270833,0 17.1111111,0 L3.11111111,0 C1.39513889,0 0,1.40122259 0,3.12467755 L0,10.9363714 C0,12.6598264 1.39513889,14.061049 3.11111111,14.061049 L3.11111111,16.7072603 C3.11111111,17.097845 3.55347222,17.3224312 3.86458333,17.0880803 L7.88958333,14.0561667 L17.1111111,14.0561667 C18.8270833,14.061049 20.2222222,12.6598264 20.2222222,10.9363714 Z M24.8888889,7.81169387 L21.7777778,7.81169387 L21.7777778,10.9363714 C21.7777778,13.5191127 19.6826389,15.6233877 17.1111111,15.6233877 L9.33333333,15.6233877 L9.33333333,18.7480653 C9.33333333,20.4715202 10.7284722,21.8727428 12.4444444,21.8727428 L18.5548611,21.8727428 L22.5798611,24.9046565 C22.8909722,25.1390073 23.3333333,24.9144211 23.3333333,24.5238364 L23.3333333,21.8727428 L24.8888889,21.8727428 C26.6048611,21.8727428 28,20.4715202 28,18.7480653 L28,10.9363714 C28,9.21291646 26.6048611,7.81169387 24.8888889,7.81169387 Z" />
          </g>
        </g>
      </g>
    </g>
  </svg>
)

export default CommentsIcon
