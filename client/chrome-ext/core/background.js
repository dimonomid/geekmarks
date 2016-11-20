//var clientInst = gmClient.create("localhost:4000", "alice", "alice");

pagesCtx = {};

function openPageWrapper(queryString) {
  chrome.windows.create({
    url: "/page-wrapper/page-wrapper.html?" + queryString,
    width: 700,
    height: 400,
    type: "popup",
  });
}

function openOrRefocusPageWrapper(portName, queryString, curTab) {
  if (!(portName in pagesCtx)) {
    console.log("opening", portName, curTab);
    pagesCtx[portName] = {
      port: undefined,
      tab: curTab,
    };
    openPageWrapper(queryString + '&port_name=' + portName);
  } else {
    console.log("refocusing", portName, curTab);
    pagesCtx[portName].tab = curTab;
    setCurTab(portName);
    pagesCtx[portName].port.postMessage({type: "cmd", cmd: "focus"});
  }
}

function setCurTab(portName) {
  pagesCtx[portName].port.postMessage(
    {type: "cmd", cmd: "setCurTab", curTab: pagesCtx[portName].tab}
  );
}

chrome.commands.onCommand.addListener(function(command) {
  //window.open("html/get-bookmark-wrapper.html", "extension_popup", "width=300,height=400,status=no,scrollbars=yes,resizable=no");
  var curTab;

  console.log("got command:", command);
  chrome.tabs.query({active: true, currentWindow: true}, function(arrayOfTabs) {

    // TODO: better check of whether it's the window of this extension
    if (arrayOfTabs[0].url.slice(6) === "chrome") {
      console.log("url chrome: ignoring")
      return;
    }

    switch (command) {
      case "query-bookmark":
        {
          // since only one tab should be active and in the current window at once
          // the return variable should only have one entry
          curTab = arrayOfTabs[0];

          openOrRefocusPageWrapper("getBookmark", "page=get-bookmark", curTab);
        }
        break;
      case "add-bookmark":
        {
          // since only one tab should be active and in the current window at once
          // the return variable should only have one entry
          curTab = arrayOfTabs[0];

          openPageAddBookmark(curTab);
        }
        break;
    }

  });

});

chrome.runtime.onConnect.addListener(
  function(port) {
    console.log("connected, port name:", port.name);

    switch (port.name) {
      //case "gmclient-bridge":
      //port.onMessage.addListener(
      //function(msg) {
      //console.log("got msg:", msg);
      //switch (msg.type) {
      //case "cmd":
      //switch (msg.cmd) {
      //case "sendViaGMClient":
      //var func = clientInst[msg.funcName];
      //msg.args.push(function(resp) {
      //port.postMessage(
      //{type: "cmd", cmd: "gmClientResp", resp: resp, id: msg.id}
      //);
      //})
      //func.apply(undefined, msg.args);
      //break;
      //}
      //break;
      //}
      //}
      //);
      //break;

      default:
        if (pagesCtx[port.name].port !== undefined) {
          throw Error("port for " + port.name + " already exists when new port connection is created");
        }

        pagesCtx[port.name].port = port;
        setCurTab(port.name);

        port.onMessage.addListener(
          function(msg) {
            console.log("got msg in port", port, ":", msg);
            switch (msg.type) {
              case "cmd":
                switch (msg.cmd) {
                  case "clearCurTab":
                    delete pagesCtx[port.name]
                    break;

                  case "openPageEditBookmarks":
                    openPageEditBookmarks(msg.bkmId, msg.curTab);
                    break;

                  case "openPageAddBookmark":
                    openPageAddBookmark(msg.curTab);
                    break;
                }
                break;
            }
          }
        );

        break;
    }

  }
);

function openPageEditBookmarks(bkmId, curTab) {
  openOrRefocusPageWrapper(
    "editBookmark-" + bkmId, "page=edit-bookmark&bkm_id=" + bkmId, curTab
  );
}

function openPageAddBookmark(curTab) {
  openOrRefocusPageWrapper(
    "addBookmark", "page=edit-bookmark&bkm_id=0", curTab
  );
}
