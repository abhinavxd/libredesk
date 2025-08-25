/**
 * Handle Axios http error response
 * @param {any} error - Axios error
 * @returns Readable error object
 */
function handleHTTPError (error) {
  let resp = {
    status: 'error',
    message: 'Unknown error',
    error_type: 'GeneralException',
    data: null,
    status_code: null
  }
  // Response received from the server.
  if (error.response && error.response.data) {
    if (error.response.data.error_type) {
      resp = error.response.data
    } else if (error.response.data.message) {
      resp.message = error.response.data.message
    }
    resp.status_code = error.response.status
  } else if (error.request) {
    resp.message = 'No response from server. Check if you are still connected to internet.'
  } else if (error.message) {
    resp.message = error.message
  } else {
    resp.message = 'Error setting up the request'
  }
  return resp
}

export { handleHTTPError }
