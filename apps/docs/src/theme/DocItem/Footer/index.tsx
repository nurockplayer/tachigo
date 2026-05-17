import React from 'react'
import DocItemFooter from '@theme-original/DocItem/Footer'

import RelatedLinks from '../../../components/RelatedLinks'

export default function Footer(props): JSX.Element {
  return (
    <>
      <RelatedLinks />
      <DocItemFooter {...props} />
    </>
  )
}
