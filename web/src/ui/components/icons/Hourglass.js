import React from 'react'

const HourglassIcon = () => (
  <svg width="32px" height="32px" viewBox="0 0 32 32" version="1.1" xmlns="http://www.w3.org/2000/svg">
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
          <g className="icon-hourglass" transform="translate(284.000000, 1.000000)">
            <path d="M22.781 16c4.305-2.729 7.219-7.975 7.219-14 0-0.677-0.037-1.345-0.109-2h-27.783c-0.072 0.655-0.109 1.323-0.109 2 0 6.025 2.914 11.271 7.219 14-4.305 2.729-7.219 7.975-7.219 14 0 0.677 0.037 1.345 0.109 2h27.783c0.072-0.655 0.109-1.323 0.109-2 0-6.025-2.914-11.271-7.219-14zM5 30c0-5.841 2.505-10.794 7-12.428v-3.143c-4.495-1.634-7-6.587-7-12.428v0h22c0 5.841-2.505 10.794-7 12.428v3.143c4.495 1.634 7 6.587 7 12.428h-22zM19.363 20.925c-2.239-1.27-2.363-2.918-2.363-3.918v-2.007c0-1 0.119-2.654 2.367-3.927 1.203-0.699 2.244-1.761 3.033-3.073h-12.799c0.79 1.313 1.832 2.376 3.036 3.075 2.239 1.27 2.363 2.918 2.363 3.918v2.007c0 1-0.119 2.654-2.367 3.927-2.269 1.318-3.961 3.928-4.472 7.073h15.677c-0.511-3.147-2.204-5.758-4.475-7.075z"></path>
          </g>
        </g>
      </g>
    </g>
  </svg>
)

export default HourglassIcon
