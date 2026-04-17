async function enableSidePanelAction() {
  if (!chrome.sidePanel?.setPanelBehavior) {
    return
  }

  try {
    await chrome.sidePanel.setPanelBehavior({ openPanelOnActionClick: true })
  } catch (error) {
    console.warn('Failed to enable side panel action behavior', error)
  }
}

chrome.runtime.onInstalled.addListener(() => {
  void enableSidePanelAction()
})

chrome.runtime.onStartup.addListener(() => {
  void enableSidePanelAction()
})

chrome.action.onClicked.addListener((tab) => {
  if (!chrome.sidePanel?.open || typeof tab.windowId !== 'number') {
    return
  }

  void chrome.sidePanel.open({ windowId: tab.windowId }).catch((error) => {
    console.warn('Failed to open side panel from action click', error)
  })
})

// Message listener stub — wired in PR 3
chrome.runtime.onMessage.addListener((_message, _sender, _sendResponse) => {
  // no-op placeholder
})
