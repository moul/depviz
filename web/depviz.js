$(document).ready(function() {
  $("#submit").click(function() {
    $("#result").html("loading JSON...");
    $.ajax({
      url: "/api/issues.json?targets=" + $("#targets").attr("value"),
      success: function(result, status, xhr) {
        console.dir(result);
        $("#result").html(JSON.stringify(result));
        $("#image-link").attr("href", "/api/graph/image?targets=" + $("#targets").attr("value"));
      },
      error: function(xhr, status, error) {
        console.error("failed", xhr, status, error);
        alert("failed: " +  error);
      },
    })
  });
});
