function localizeDatetime(datetimeStr) {
  var date = new Date(datetimeStr)
  return date.toLocaleString('uk-UA', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function localizeDate(dateStr) {
  var date = new Date(dateStr)
  return date.toLocaleString('uk-UA', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  })
}

