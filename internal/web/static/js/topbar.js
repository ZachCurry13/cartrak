// On a computer the tabs move into the header, so the name, the car picker, the tabs, the car's status and the icons share one row
// instead of two bars stacked on top of each other. A wider window also gets more tabs: the first pages of the More sheet. On a
// phone or a narrow window nothing changes: the tabs stay at the bottom.

const ONE_ROW = matchMedia('(min-width:1280px)');
const MORE_TABS = matchMedia('(min-width:1700px)');

// How many pages of the More sheet join the tabs at this width.
export const extraTabs = () => (MORE_TABS.matches ? 3 : 0);

export function mountTopbar(onWidth) {
  const head = document.getElementById('top'), tabs = document.getElementById('tabs');
  if (!head || !tabs) return;
  const home = document.createComment('tabs'); // where the tabs live on a phone
  tabs.before(home);
  const place = () => {
    if (ONE_ROW.matches && tabs.parentElement !== head) head.insertBefore(tabs, head.querySelector('.hicons'));
    else if (!ONE_ROW.matches && tabs.parentElement === head) home.after(tabs);
    document.documentElement.classList.toggle('onerow', ONE_ROW.matches);
  };
  place();
  ONE_ROW.addEventListener('change', place);
  MORE_TABS.addEventListener('change', () => onWidth());
}
