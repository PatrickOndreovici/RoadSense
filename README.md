# RoadSense

RoadSense reconstructs the most likely route traveled from raw GPS coordinates using OpenStreetMap road data.

<p align="center">
  <img src="assets/demo.gif" alt="RoadSense Demo" width="850">
</p>

Instead of connecting GPS points with straight lines, the application snaps them to the road network and computes the most realistic path between them.

### Built with

- Go
- React
- OpenStreetMap

### Under the hood

- Viterbi-based map matching
- Dijkstra shortest path
- R-Tree spatial indexing