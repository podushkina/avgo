import type { ReactNode } from 'react';
import { createPortal } from 'react-dom';

import { HEADER_RIGHT_SECTION_ID } from '../constants';

type HeaderRightSectionPortalProps = {
  children: ReactNode;
};

const HeaderRightSectionPortal = ({
  children,
}: HeaderRightSectionPortalProps) => {
  const container = document.getElementById(HEADER_RIGHT_SECTION_ID);

  if (!container) {
    return null;
  }

  return createPortal(children, container);
};

export default HeaderRightSectionPortal;
