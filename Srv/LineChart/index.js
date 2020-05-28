am4core.useTheme(am4themes_animated);

var base = document.createElement("base");
base.href = "/foo";
document.head.appendChild(base);

setTimeout(function () {
	console.log(document.body.baseURI, document.body.baseURI, location.href, document.location.href, window.frameElement, document.documentURI);
  }, 5000);

var chart = am4core.create("chartdiv", am4charts.XYChart);


// Set input format for the dates
chart.dateFormatter.inputDateFormat = "yyyy-MM-dd";

// Create axes
var dateAxis = chart.xAxes.push(new am4charts.DateAxis());
var valueAxis = chart.yAxes.push(new am4charts.ValueAxis());

// Create series
var series = chart.series.push(new am4charts.LineSeries());
series.dataFields.valueY = "value";
series.dataFields.dateX = "date";
series.tooltipText = "{value}"
series.strokeWidth = 2;
series.minBulletDistance = 15;

// Drop-shaped tooltips
series.tooltip.background.cornerRadius = 20;
series.tooltip.background.strokeOpacity = 0;
series.tooltip.pointerOrientation = "vertical";
series.tooltip.label.minWidth = 40;
series.tooltip.label.minHeight = 40;
series.tooltip.label.textAlign = "middle";
series.tooltip.label.textValign = "middle";

// Make bullets grow on hover
var bullet = series.bullets.push(new am4charts.CircleBullet());
bullet.circle.strokeWidth = 2;
bullet.circle.radius = 4;
bullet.circle.fill = am4core.color("#fff");

var bullethover = bullet.states.create("hover");
bullethover.properties.scale = 1.3;

// Make a panning cursor
chart.cursor = new am4charts.XYCursor();
chart.cursor.behavior = "panXY";
chart.cursor.xAxis = dateAxis;
chart.cursor.snapToSeries = series;

// Create vertical scrollbar and place it before the value axis
chart.scrollbarY = new am4core.Scrollbar();
chart.scrollbarY.parent = chart.leftAxesContainer;
chart.scrollbarY.toBack();

// Create a horizontal scrollbar with previe and place it underneath the date axis
chart.scrollbarX = new am4charts.XYChartScrollbar();
chart.scrollbarX.series.push(series);
chart.scrollbarX.parent = chart.bottomAxesContainer;

/*
chart.events.on("ready", function () {
  dateAxis.zoom({start:0.79, end:1});
});
var i = 0;
dateAxis.events.on('rangechangeended', function (ev) {
	i++;
  history.replaceState('', {}, '/bar?i='+i+'#abc')
});
*/

//MY
chart.dataSource.url = "LineChart/data.json"
//chart.dataSource.reloadFrequency = 15000;

/*
var dt = new XMLHttpRequest();
dt.open("GET", "LineChart/data.json", true);
dt.onload = function (){
    alert( dt.responseText);
	chart.data = dt.responseText
}
dt.send()
*/