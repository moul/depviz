import html2canvas from 'html2canvas'

const exportCanvasToImage = (selector, appendTo) => html2canvas(selector)

export default exportCanvasToImage
