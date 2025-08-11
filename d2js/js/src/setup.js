console.log("setup.js: Starting execution");
var setTimeout = function (f) {
  f();
};
console.log("setup.js: ELK available:", typeof ELK);
var elk = new ELK();
console.log("setup.js: Created elk instance:", typeof elk);
globalThis.elk = elk;
console.log("setup.js: Set globalThis.elk:", typeof globalThis.elk);
console.log("setup.js: Finished execution");
