import { isFuelCar } from './navlayout.js';

// The few lines of explanation that name charging or the battery, said the way the car in front of you needs: a gas or hybrid car
// gets fuel wording and no EV figures (kWh, charging stops, plugging in).
export function carWords(session) {
  const fuel = isFuelCar(session);
  return {
    fuel,
    plan: fuel ? 'How many fuel stops a trip needs' : 'How many charging stops a trip needs',
    insights: fuel ? 'What changes your miles per gallon' : 'What changes your miles per kWh',
    insightsLong: fuel ? 'What changes your miles per gallon: weather, speed, habits' : 'What changes your miles per kWh: weather, speed, habits',
    year: fuel ? 'Miles and fuel for the year' : 'Miles, energy and charging for the year',
    history: fuel ? 'Driving and parked time as one feed' : 'Driving, charging and parked time as one feed',
    stats: fuel ? 'Bars by day, month and year' : 'Bars by day, month and year, and where you charge',
    time: fuel ? 'Parked and driving, and where' : 'Parked, driving and charging, and where',
    alerts: fuel ? 'Low fuel, a low tire, a service due and more.' : 'Charging done, plug in tonight, a low tire and more.',
  };
}
