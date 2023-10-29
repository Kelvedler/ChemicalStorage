function subscriptNumbers(e){
  const replaceMap = new Map([
    ["0", "\u2080"],
    ["1", "\u2081"],
    ["2", "\u2082"],
    ["3", "\u2083"],
    ["4", "\u2084"],
    ["5", "\u2085"],
    ["6", "\u2086"],
    ["7", "\u2087"],
    ["8", "\u2088"],
    ["9", "\u2089"],
  ]);
  if (e.ctrlKey === true && /^[0-9]$/i.test(e.key)) {
    e.explicitOriginalTarget.value += replaceMap.get(e.key);
  }
}
