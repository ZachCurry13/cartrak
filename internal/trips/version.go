package trips

// DetectionVersion goes up by one whenever a change to how trips and charging are found would record an old day
// differently if it were redone. Home then offers Fix a day once; otherwise Fix a day stays out of sight.
const DetectionVersion = 2
