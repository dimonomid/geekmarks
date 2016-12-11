'use strict';

(function(exports){

  var contentElem = undefined;

  function init(gmClient, _contentElem, srcDir, queryParams, curTabData) {
    contentElem = _contentElem;
    var tagsTreeDiv = contentElem.find('#tags_tree_div')

    gmClient.getTagsTree(function(status, resp) {
      if (status == 200) {
        var treeData = convertTreeData(resp);

        tagsTreeDiv.fancytree({
          extensions: ["edit"],
          edit: {
            adjustWidthOfs: 4,   // null: don't adjust input size to content
            inputCss: { minWidth: "3em" },
            triggerStart: ["f2", "dblclick", "shift+click", "mac+enter"],
            beforeEdit: $.noop,  // Return false to prevent edit mode
            edit: $.noop,        // Editor was opened (available as data.input)
            beforeClose: $.noop, // Return false to prevent cancel/save (data.input is available)
            save: saveTag,       // Save data.input.val() or return false to keep editor open
            close: $.noop,       // Editor was removed
          },
          source: treeData.children,
        });
      } else {
        // TODO: show error
        alert(JSON.stringify(resp));
      }
    })
  }

  function convertTreeData(tagsTree) {
    var ret = {
      title: tagsTree.names.join(","),
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
    var val = data.input.val();
    // TODO: save value
  }

  exports.init = init;

})(typeof exports === 'undefined' ? this['gmTagsTree']={} : exports);
