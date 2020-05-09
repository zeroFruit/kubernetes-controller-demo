if (process.argv.length < 4) {
  throw new Error('logger name and timeInterval must be given.');
}

const name = process.argv[2];
const timeInterval = parseInt(process.argv[3]);

console.log('Logger Name: ', name);
console.log('Logger TimeInterval: ', timeInterval);

const logger = setInterval(function() {
  console.log('Logging user activity ...', new Date());
}, timeInterval);

process.on('exit', code => {
  console.log('process exit');
  clearInterval(logger);
});
