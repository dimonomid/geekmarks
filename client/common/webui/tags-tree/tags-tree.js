'use strict';

(function(exports){

  var contentElem = undefined;
  var gmClient = undefined;

  function init(_gmClient, _contentElem, srcDir, queryParams, curTabData) {
    contentElem = _contentElem;
    gmClient = _gmClient;
    var tagsTreeDiv = contentElem.find('#tags_tree_div')

    gmClient.getTagsTree(function(status, resp) {
      if (status == 200) {
        var treeData = convertTreeData(resp);

        tagsTreeDiv.fancytree({
          extensions: ["edit", "table"],
          edit: {
            adjustWidthOfs: 4,   // null: don't adjust input size to content
            inputCss: { minWidth: "3em" },
            triggerStart: ["f2", "shift+click", "mac+enter"],
            beforeEdit: $.noop,  // Return false to prevent edit mode
            edit: $.noop,        // Editor was opened (available as data.input)
            beforeClose: $.noop, // Return false to prevent cancel/save (data.input is available)
            save: saveTag,       // Save data.input.val() or return false to keep editor open
            close: $.noop,       // Editor was removed
          },
          table: {
          },
          source: treeData.children,
          renderColumns: function(event, data) {
            var node = data.node;
            var $tdList = $(node.tr).find(">td");
            var $ctrlCol = $tdList.eq(2);
            $ctrlCol.text("");
            $("<a/>", {
              href: "#",
              text: "[edit]",
              click: function() {
                gmPageWrapper.openPageEditTag(data.node.key);
              },
            }).appendTo($ctrlCol);
          },
        });
      } else {
        // TODO: show error
        alert(JSON.stringify(resp));
      }
    })
  }

  function convertTreeData(tagsTree) {
    var ret = {
      title: tagsTree.names.join(", "),
      key: tagsTree.id,
    };
    if ("subtags" in tagsTree) {
      ret.children = tagsTree.subtags.map(function(a) {
        return convertTreeData(a);
      });
      ret.folder = true;
    }
    return ret;
  }

  // see https://github.com/mar10/fancytree/wiki/ExtEdit for argument details
  function saveTag(event, data) {
    $(data.node.span).addClass("pending");
    var val = data.input.val();
    var prevVal = data.orgTitle;
    //console.log('saveTag event', event)
    //console.log('saveTag data', data)

    gmClient.updateTag(String(data.node.key), {
      names: val.split(",").map(function(a) {
        return a.trim();
      }),
    }, function(status, resp) {
      if (status == 200) {
        // update succeeded, do nothing here
      } else {
        // TODO: show error
        alert(JSON.stringify(resp));
        data.node.setTitle(prevVal);
      }

      $(data.node.span).removeClass("pending");
    });

    // Optimistically assume that save will succeed. Accept the user input
    return true;
  }

  exports.init = init;

})(typeof exports === 'undefined' ? this['gmTagsTree']={} : exports);
